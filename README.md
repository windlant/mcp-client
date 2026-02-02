○完成了mcp-client，能初步作为单agent使用，实现了上下文窗口管理，tools调用功能

○并设计了统一的tools_client接口以调用mcp-server，在此接口上实现了两种mcp-server（client本地集成工具调用、子进程mcp-server通过stdio调用），预留了http远程调用mcp-server接口

○基于go-http实现与Deepseek的数据传输，并根据MCP设计了相关数据结构完成对mcp-server的工具调用
