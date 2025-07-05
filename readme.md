# 🧠 forgor

**LLM-powered command line memory assistant**

> Transform natural language into shell commands with AI assistance

`forgor` is a CLI tool that translates your natural language descriptions into executable shell commands using Large Language Models. Whether you forgot a specific syntax or need help crafting complex commands, `forgor` has your back.

---

## ✨ Features

- 🤖 **Multiple LLM Providers**: OpenAI, Anthropic Claude, Google Gemini
- 🔧 **Flexible Configuration**: Profile-based setup with environment variable support
- 📚 **Shell History Integration**: Context-aware suggestions using command history
- 🎯 **Smart Context Detection**: Automatically detects your OS, shell, and available tools
- 🛡️ **Safety Features**: Danger assessment and warnings for potentially destructive commands
- 🔄 **Interactive Mode**: Follow-up questions and command refinement
- 📖 **Explain Mode**: Get detailed explanations of what commands do
- ⚡ **Shell Completion**: Tab completion for all major shells (bash, zsh, fish)
- 🏃 **Force Run Mode**: Directly execute generated commands (use with caution)

---

### Enhanced Shell History (Recommended)

For the best experience, `forgor` can use an enhanced shell logger that provides rich, real-time, cross-session history.

This logger provides more context than native shell history:

- Captures commands from all terminal sessions instantly.
- Includes context like the working directory for each command.
- Sanitizes sensitive arguments (e.g., passwords, API keys) automatically through a filter list.

In saying this, please note that the enhanced logger is not a replacement for native shell history. and although it itself doesn't send commands to an external API, it can still be sent to LLMs so do be aware of the potential risks of any keys or secrets you might have in your history.

You can disable history completely by setting `history: 0` in your configuration file.

#### Install the Enhanced Logger

Run the following command to install the logger script. It will automatically detect your shell (`bash`, `zsh`, or `fish`) and configure it.

```bash
curl -sL https://raw.githubusercontent.com/Siutan/forgor/main/scripts/configure-history-logger.sh | bash
```

After running, **restart your shell** to activate it. The logger does not overwrite your history file, it hooks into the `command has run` event and logs each command to the `~/.command_log` file.

#### Uninstall the Logger

If you need to remove the logger, a simple uninstall script is provided:

```bash
curl -sL https://raw.githubusercontent.com/Siutan/forgor/main/scripts/reset-history-logger.sh | bash
```

---

## 🚀 Installation

### Quick Install

```bash
  curl -fsSL https://raw.githubusercontent.com/Siutan/forgor/refs/heads/main/scripts/install.sh | sh
```

### Add to PATH

If the install script doesn't automatically add the binary to your PATH, add it manually

### Create Alias (Recommended)

For convenience, create an alias to use `ff` instead of typing `forgor`:

```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
alias ff="forgor"

# Source your profile
source ~/.bashrc  # or ~/.zshrc, ~/.bash_profile, etc.
```

### Verify Installation

```bash
forgor --version
# or with alias
ff --version
```

---

## ⚙️ Setup

### 1. Initialize Configuration

```bash
forgor config init
```

This creates a default configuration at `~/.config/forgor/config.yaml`.

### 2. Set API Keys

Configure your preferred LLM provider by setting environment variables:

```bash
# For OpenAI
export OPENAI_API_KEY="your-api-key-here"

# For Anthropic Claude
export ANTHROPIC_API_KEY="your-api-key-here"

# For Google Gemini
export GOOGLE_AI_API_KEY="your-api-key-here"
```

Add these to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to persist them.

### 3. Set Default Provider

```bash
forgor config set-default openai      # or anthropic, gemini
```

### 4. Setup Shell Completion (Optional)

```bash
forgor config completion              # Auto-detect shell
# or specify shell explicitly
forgor config completion zsh
```

---

## 🧾 Usage

### Basic Usage

```bash
# Basic command generation
forgor "find all txt files containing 'hello'"

# With alias (if configured)
ff "show me how to make a new tmux session called dev"
```

### History-Aware Commands

```bash
# Include last 2 commands from history for context
forgor --history 2 "fix the above command"

# Short form
forgor -n 1 "make the last command safer"
```

### Different Modes

```bash
# Explain what a command does
forgor --explain "docker rm -f \$(docker ps -aq)"

# Interactive mode with follow-ups
forgor --interactive "help me set up a web server"

# Force run the generated command (DANGEROUS - use carefully)
forgor --force-run "list all files in current directory"
```

### Using Different Providers

```bash
# Use a specific provider profile
forgor --profile anthropic "optimize this bash script"

# Check available providers
forgor config list-providers
```

---

## 📋 Examples

### File Operations

```bash
forgor "recursively find all Python files modified in the last 7 days"
# Output: find . -name "*.py" -mtime -7

forgor "compress the current directory excluding node_modules"
# Output: tar --exclude='node_modules' -czf archive.tar.gz .
```

### Docker Commands

```bash
forgor "stop all running containers"
# Output: docker stop $(docker ps -q)

forgor "remove all unused docker images"
# Output: docker image prune -a
```

### System Administration

```bash
forgor "show processes using port 8080"
# Output: lsof -i :8080

forgor "find large files over 100MB in home directory"
# Output: find ~ -type f -size +100M
```

### Git Operations

```bash
forgor "undo the last commit but keep changes"
# Output: git reset --soft HEAD~1

forgor "show git log in one line format"
# Output: git log --oneline
```

---

## 🔧 Configuration

### Configuration File

The configuration file is located at `~/.config/forgor/config.yaml`:

```yaml
default_profile: "openai"

profiles:
  openai:
    provider: "openai"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
    max_tokens: 150
    temperature: 0.1

  anthropic:
    provider: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-sonnet-20240229"
    max_tokens: 150
    temperature: 0.1

  gemini:
    provider: "gemini"
    api_key: "${GOOGLE_AI_API_KEY}"
    model: "gemini-1.5-flash"
    max_tokens: 150
    temperature: 0.1

history:
  max_commands: 10
  shells: ["bash", "zsh", "fish"]

security:
  redact_sensitive: true
  filters:
    - "password"
    - "token"
    - "secret"
    - "key"

output:
  format: "plain"
```

### Configuration Commands

```bash
# Show current configuration
forgor config show

# Set default provider
forgor config set-default anthropic

# List all available providers
forgor config list-providers
```

---

## 🛡️ Safety Features

`forgor` includes built-in safety features to protect against dangerous commands:

- **Danger Assessment**: Commands are analyzed for potential risks
- **Warning System**: Destructive operations trigger warnings
- **Confirmation Prompts**: High-risk commands require explicit confirmation
- **Sensitive Data Filtering**: API keys and passwords are filtered from prompts

---

## 🗺️ Roadmap

### ✅ Implemented

- [x] Core LLM integration (OpenAI, Anthropic, Gemini)
- [x] Configuration management with profiles
- [x] Shell history integration
- [x] Context-aware prompting
- [x] Interactive and explain modes
- [x] Shell completion
- [x] Safety and danger assessment
- [x] Force run mode
- [x] Comprehensive configuration tools

### 🚧 In Progress

- [ ] Additional LLM providers (Ollama, local models)
- [ ] Enhanced history filtering and search
- [ ] Command templates and favorites
- [ ] Plugin system for custom providers

### 🔮 Future

- [ ] Web interface for command history
- [ ] Team collaboration features
- [ ] Advanced prompt engineering tools
- [ ] Integration with external documentation
- [ ] Command learning and suggestions

---

## 🧑‍💻 Development

### Building from Source

```bash
git clone https://github.com/Siutan/forgor.git
cd forgor
make build
```

### Running Tests

```bash
go test ./...
```

For detailed development information, see the [Development Guide](docs/DEVELOPMENT.md).

Also the ci tests fail on windows, but i use a mac so i cant really fix it. If you know how to fix it, please do.

---

## 🤝 Contributing

We welcome contributions! Please see our [Development Guide](docs/DEVELOPMENT.md) for detailed information on:

- Setting up the development environment
- Code quality standards
- Creating pull requests
- Testing procedures

---

## 🔐 Privacy & Security

- **API Keys**: Never logged or exposed in output
- **Sensitive Data**: Automatically filtered from prompts (basic keyword detection)
- **Local Mode**: Option to avoid external API calls
- **Command Validation**: Built-in safety checks for dangerous operations

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 🆘 Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/Siutan/forgor/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/Siutan/forgor/discussions)
- 📧 **Email**: [Contact the maintainers](https://github.com/Siutan/forgor#maintainers)

---

_"Never forget how to command your terminal again."_
