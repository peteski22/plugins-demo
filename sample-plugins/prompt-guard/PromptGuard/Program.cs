using System.Text.Json;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using Microsoft.Extensions.Logging;
using MozillaAI.Mcpd.Plugins.V1;

/// <summary>
/// Prompt guard plugin that scans request JSON bodies for blocked phrases.
/// </summary>
public class PromptGuard : BasePlugin
{
    private static readonly string[] BlockedPhrases =
    [
        "naughty naughty very naughty",
        "ignore previous instructions",
        "disregard",
        "system prompt",
        "you are now"
    ];

    private static readonly string name = "plugin-prompt-guard";

    private bool _initialized = false;

    public override Task<Metadata> GetMetadata(Empty request, Grpc.Core.ServerCallContext context)
    {
        return Task.FromResult(new Metadata
        {
            Name = name,
            Version = "1.0.0",
            Description = "Scans request JSON bodies for blocked phrases (SDK version)"
        });
    }

    public override Task<Capabilities> GetCapabilities(Empty request, Grpc.Core.ServerCallContext context)
    {
        return Task.FromResult(new Capabilities
        {
            Flows = { FlowConstants.FlowRequest }
        });
    }

    public override Task<Empty> Configure(PluginConfig request, Grpc.Core.ServerCallContext context)
    {
        _initialized = true;
        Logger.LogInformation("Prompt guard plugin configured");
        return Task.FromResult(new Empty());
    }

    public override Task<Empty> Stop(Empty request, Grpc.Core.ServerCallContext context)
    {
        _initialized = false;
        Logger.LogInformation("Prompt guard plugin stopped");
        return Task.FromResult(new Empty());
    }

    public override Task<Empty> CheckHealth(Empty request, Grpc.Core.ServerCallContext context)
    {
        if (!_initialized)
        {
            throw new Grpc.Core.RpcException(new Grpc.Core.Status(Grpc.Core.StatusCode.FailedPrecondition, "Plugin not initialized"));
        }

        return Task.FromResult(new Empty());
    }

    public override Task<Empty> CheckReady(Empty request, Grpc.Core.ServerCallContext context)
    {
        if (!_initialized)
        {
            throw new Grpc.Core.RpcException(new Grpc.Core.Status(Grpc.Core.StatusCode.FailedPrecondition, "Plugin not ready"));
        }

        return Task.FromResult(new Empty());
    }

    public override Task<HTTPResponse> HandleRequest(HTTPRequest request, Grpc.Core.ServerCallContext context)
    {
        Logger.LogInformation("Prompt guard handling request: {Method} {Path}", request.Method, request.Path);

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
                Logger.LogWarning("Blocked phrase detected: {Phrase}", foundPhrase);

                var errorJson = JsonSerializer.Serialize(new
                {
                    error = "Request blocked: prohibited content detected",
                    reason = $"Phrase '{foundPhrase}' is not allowed",
                    plugin = name
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

public class Program
{
    public static async Task<int> Main(string[] args)
    {
        return await PluginServer.Serve<PromptGuard>(args);
    }
}
