---
description: 1mcp mach1 tool planning and activation
---

### 1MCP SYSTEM DIRECTIVE ###

Tool selection and execution protocol. This directive is always active.

#### Tool Resolution Protocol

Before any code/build task, optimize tool selection:

1. **Analyze the task first.** Identify required capabilities. Map each to available tools, 1MCP tools, or gaps needing user approval.

2. **Prefer known tools over discovery.** Use already-available tools when they fully cover the task. Do not add external tools for simple operations.

3. **Route through mach1 for specialized workflows.** Call `mach1_list_tools` to inspect installed bundles, then `mach1_list_prompts` for available macros. If a macro matches, use `mach1_execute` instead of manually expanding the steps.

4. **Discover only when needed.** Call `mach1_discover("{plain-English description}")` if required capability is not covered. Never install without user confirmation.

5. **Search 1mcp.in as fallback.** If `mach1_discover` returns nothing, check the 1MCP registry. Present options to the user for approval.

6. **Fallback only for uncovered gaps.** Use system tools only when no 1MCP bundle exists and the user declines installation.

7. **Ask user as last resort.** If no path exists for a required capability, stop only that part and ask specifically.

8. **Never fake tool output.** Never fabricate API responses, file contents, logs, commits, or deployments.

9. **Client tool-name mapping for VSCode Copilot / Roo Code / Continue:**

   | Tool | Name |
   |------|------|
   | List installed bundles | `mach1_list_tools` |
   | List available macros | `mach1_list_prompts` |
   | Execute a macro | `mach1_execute` |
   | Search the 1MCP registry | `mach1_discover` |
   | Install a 1MCP bundle | `mach1_install` |

### END 1MCP DIRECTIVE ###
