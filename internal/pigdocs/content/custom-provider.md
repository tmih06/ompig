# Custom providers

A Ompig extension can register or override model providers. Use this mechanism when an OpenAI-compatible `models.json` entry is not sufficient or when the provider needs custom authentication, refresh, or request behavior.

Provider registration is an extension capability. Keep provider-specific code outside Stock Ompig.

## Choose the smallest mechanism

| Need | Use |
|---|---|
| Add an OpenAI-compatible endpoint and static models | `~/.ompig/agent/models.json` |
| Add provider metadata or OAuth | Provider extension |
| Add custom request or stream behavior | Provider extension |
| Select the provider for one agent application | Ompiglet `model` preference plus the provider extension |
| Supply product credentials or endpoints | External product configuration, not portable Ompiglet source |

See [Models](/docs/latest/models) before you create a provider extension.

## Register a provider

Create a Go extension:

```bash
ompig extension init ./corporate-ai --lang go
```

Register provider configuration from its factory:

```go
package corporateai

import sdk "github.com/tmih06/ompig/extensions/sdk"

func Extension() *sdk.Extension {
    ext := sdk.New("corporate-ai")
    ext.RegisterProvider("corporate-ai", sdk.ProviderConfig{
        "name":    "Corporate AI",
        "baseUrl": "https://ai.example.com/v1",
        "apiKey":  "$CORPORATE_AI_API_KEY",
        "api":     "openai-completions",
    })
    return ext
}
```

Use environment interpolation for a secret. Do not place a literal credential in extension source or Ompiglet YAML.

The exact provider model fields depend on the provider API implemented by the current Ompig version. Validate the extension against the binary that will run it.

## Validate registration

```bash
ompig install ./corporate-ai --validate-only --json
```

List models after selecting the extension:

```bash
ompig -e ./corporate-ai --list-models
```

Use a Ompiglet when the provider belongs to a named application:

```yaml
name: corporate-review
extensions:
  - name: corporate-ai
    origins: [local:./extensions/corporate-ai]
model:
  provider: corporate-ai
  name: review-model
```

## OAuth providers

An extension can register an OAuth provider through the same provider declaration. The language SDK exposes callbacks for:

- browser authorization;
- device codes;
- progress messages;
- text prompts;
- option selection;
- manual code input;
- token refresh;
- API-key derivation;
- optional extension-owned credential storage.

Ompig inspects registration before a Session so the provider can appear in:

```bash
ompig login --list
ompig login corporate-ai
```

Registration inspection constructs the extension but does not dispatch Session, tool, command, event, shortcut, or renderer handlers.

Duplicate provider IDs fail and identify both owners. Login starts only the extension that owns the selected provider.

## Credential ownership

Use Ompig's normal credential store when the provider does not declare its own store. Use an extension-owned store only when the provider contract requires it.

An extension-owned store must:

- protect file permissions;
- avoid logging tokens;
- report whether credentials are present;
- delete credentials through the matching logout path;
- surface errors instead of returning false success.

Environment variables take precedence when the provider contract defines that behavior.

## Dynamic provider state

Provider registrations belong to the live extension generation. Ompig removes them when the extension shuts down or a replacement fails to register.

A successful atomic reload replaces provider state with the new validated extension set. A failed replacement keeps the previous working set.

## Connection failure

A provider extension can hold live state even when no tool call is active. Ompig keeps heartbeat active while the connection owns that state.

If the connection fails, Ompig cancels pending calls and removes state owned by the failed generation. It does not replay a provider request automatically.

## Fused providers

A compatible Go provider extension can be fused into a Ompiglet Binary. Fusion does not change registration or credential semantics.

Ompig Standard requires all selected extensions to fuse. A provider added to Standard must therefore be a compatible Go factory and pass source and fused lifecycle tests.

## Security

A provider extension can receive prompts, model responses, headers, and credentials. Review it as security-sensitive code.

Do not:

- hardcode credentials;
- send Ompig or user data to an undeclared endpoint;
- hide certificate failures;
- retry non-idempotent requests without a defined policy;
- retain tokens after logout;
- claim an upstream provider identity for a different service.

## Upstream distinction

Upstream Pi exposes its provider API to in-process TypeScript extensions. Ompig maps equivalent behavior through its extension SDKs and host contract.

Use [upstream Pi's custom provider documentation](https://pi.dev/docs/latest/custom-provider) when you are writing an extension for Pi itself.

## Related documentation

- [Models](/docs/latest/models)
- [Providers](/docs/latest/providers)
- [Extensions](/docs/latest/extensions)
- [Ompiglets](/docs/latest/ompiglets)
