using System.Net;
using System.Text.Json;
using Google.Protobuf;
using Grpc.Core;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Server.Kestrel.Core;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using MozillaAI.Mcpd.Plugins.V1;

namespace PromptGuard;

class PromptGuardPlugin : Plugin.PluginBase
{
    private static readonly string[] BlockedPhrases =
    [
        "naughty naughty very naughty",
        "ignore previous instructions",
        "disregard",
        "system prompt",
        "you are now"
    ];

    private bool _initialized = false;

    public override Task<MozillaAI.Mcpd.Plugins.V1.Metadata> GetMetadata(Google.Protobuf.WellKnownTypes.Empty request, ServerCallContext context)
    {
        return Task.FromResult(new MozillaAI.Mcpd.Plugins.V1.Metadata
        {
            Name = "prompt-guard",
            Version = "1.0.0",
            Description = "Scans request JSON bodies for blocked phrases and rejects requests containing them"
        });
    }

    public override Task<Capabilities> GetCapabilities(Google.Protobuf.WellKnownTypes.Empty request, ServerCallContext context)
    {
        return Task.FromResult(new Capabilities
        {
            Flows = { Flow.Request }
        });
    }

    public override Task<Google.Protobuf.WellKnownTypes.Empty> Configure(PluginConfig request, ServerCallContext context)
    {
        _initialized = true;
        Console.WriteLine("Prompt guard plugin configured");
        return Task.FromResult(new Google.Protobuf.WellKnownTypes.Empty());
    }

    public override Task<Google.Protobuf.WellKnownTypes.Empty> Stop(Google.Protobuf.WellKnownTypes.Empty request, ServerCallContext context)
    {
        _initialized = false;
        Console.WriteLine("Prompt guard plugin stopped");
        return Task.FromResult(new Google.Protobuf.WellKnownTypes.Empty());
    }

    public override Task<Google.Protobuf.WellKnownTypes.Empty> CheckHealth(Google.Protobuf.WellKnownTypes.Empty request, ServerCallContext context)
    {
        if (!_initialized)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Plugin not initialized"));
        }

        return Task.FromResult(new Google.Protobuf.WellKnownTypes.Empty());
    }

    public override Task<Google.Protobuf.WellKnownTypes.Empty> CheckReady(Google.Protobuf.WellKnownTypes.Empty request, ServerCallContext context)
    {
        if (!_initialized)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Plugin not ready"));
        }

        return Task.FromResult(new Google.Protobuf.WellKnownTypes.Empty());
    }

    public override Task<HTTPResponse> HandleRequest(HTTPRequest request, ServerCallContext context)
    {
        Console.WriteLine($"Prompt guard handling request: {request.Method} {request.Path}");

        if (request.Body is null or { IsEmpty: true })
        {
            return Task.FromResult(new HTTPResponse { Continue = true });
        }

        try
        {
            var bodyString = request.Body.ToStringUtf8();
            var jsonDoc = JsonDocument.Parse(bodyString);

            if (ScanJsonElement(jsonDoc.RootElement, out var foundPhrase))
            {
                Console.WriteLine($"Blocked phrase detected: {foundPhrase}");

                var errorJson = JsonSerializer.Serialize(new
                {
                    error = "Request blocked: prohibited content detected",
                    reason = $"Phrase '{foundPhrase}' is not allowed"
                });

                return Task.FromResult(new HTTPResponse
                {
                    Continue = false,
                    StatusCode = 400,
                    Body = ByteString.CopyFromUtf8(errorJson),
                    Headers =
                    {
                        { "Content-Type", "application/json" }
                    }
                });
            }
        }
        catch (JsonException)
        {
            // Not valid JSON, let it pass through.
        }

        return Task.FromResult(new HTTPResponse { Continue = true });
    }

    public override Task<HTTPResponse> HandleResponse(HTTPResponse request, ServerCallContext context)
    {
        return Task.FromResult(new HTTPResponse
        {
            Continue = true,
            StatusCode = request.StatusCode,
            Headers = { request.Headers },
            Body = request.Body
        });
    }

    private bool ScanJsonElement(JsonElement element, out string foundPhrase)
    {
        foundPhrase = string.Empty;

        switch (element.ValueKind)
        {
            case JsonValueKind.String:
                var value = element.GetString() ?? string.Empty;
                foreach (var phrase in BlockedPhrases)
                {
                    if (value.Contains(phrase, StringComparison.OrdinalIgnoreCase))
                    {
                        foundPhrase = phrase;
                        return true;
                    }
                }
                break;

            case JsonValueKind.Array:
                foreach (var item in element.EnumerateArray())
                {
                    if (ScanJsonElement(item, out foundPhrase))
                    {
                        return true;
                    }
                }
                break;

            case JsonValueKind.Object:
                foreach (var property in element.EnumerateObject())
                {
                    if (ScanJsonElement(property.Value, out foundPhrase))
                    {
                        return true;
                    }
                }
                break;
        }

        return false;
    }
}

class Program
{
    static async Task Main(string[] args)
    {
        try
        {
            string? address = null;
            string network = "unix";

            for (int i = 0; i < args.Length; i++)
            {
                if (args[i] == "--address" && i + 1 < args.Length)
                {
                    address = args[++i];
                }
                else if (args[i] == "--network" && i + 1 < args.Length)
                {
                    network = args[++i];
                }
            }

            if (string.IsNullOrEmpty(address))
            {
                await Console.Error.WriteLineAsync("Error: --address flag is required");
                Environment.Exit(1);
            }

            var builder = WebApplication.CreateBuilder(args);
            builder.Logging.ClearProviders();

            builder.WebHost.ConfigureKestrel(options =>
            {
                if (network == "tcp")
                {
                    var parts = address.Split(':');
                    var host = parts.Length > 1 ? parts[0] : "127.0.0.1";
                    var port = parts.Length > 1 ? int.Parse(parts[1]) : int.Parse(address);

                    options.Listen(IPAddress.Parse(host), port, listenOptions =>
                    {
                        listenOptions.Protocols = HttpProtocols.Http2;
                    });
                }
                else
                {
                    options.ListenUnixSocket(address, listenOptions =>
                    {
                        listenOptions.Protocols = HttpProtocols.Http2;
                    });
                }
            });

            builder.Services.AddGrpc();

            var app = builder.Build();
            app.MapGrpcService<PromptGuardPlugin>();

            await app.StartAsync();
            Console.WriteLine($"Prompt guard plugin listening on {network} {address}");

            await app.WaitForShutdownAsync();
        }
        catch (Exception ex)
        {
            await Console.Error.WriteLineAsync($"Fatal error in prompt-guard plugin: {ex.Message}");
            await Console.Error.WriteLineAsync($"Stack trace: {ex.StackTrace}");
            Environment.Exit(1);
        }
    }
}
