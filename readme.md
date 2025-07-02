# 🧠 forgor

> _"bro how do I grep again?"_ > `forgor` is your LLM-powered memory jogger for the command line. You describe what you **meant to do**, and `forgor` helps you **remember** how to do it.

---

## 🧾 What is this?

`forgor` (`ff`) is a CLI tool that lets you type **natural language prompts** and get **bash-friendly commands** back, powered by an LLM (local or remote, you decide). It's like having a shell-savvy bredrin on standby, no matter how long ago you last used `awk`.

You talk. It translates.

---

## 🧪 Examples

```bash
$ ff select all txt files with the string "hello" in it
# Output:
grep -rl "hello" --include="*.txt" .

$ ff show me how to make a new tmux session called dev
# Output:
tmux new -s dev

$ ff how do I kill all docker containers
# Output:
docker rm -f $(docker ps -aq)
```

### 🕰️ History-powered fixes

Let's say you mess up a command or get an error. You can fix it with context from the past:

```bash
$ ff -h 1 fix the above command
```

> Looks at the last command in history (`-h 1`) and sends that + your fix request to the LLM.

```bash
$ ff -h 3 make the third command safer
```

---

## 🔧 Installation

_This is placeholder until you build it fam, but pattern it like this:_

```bash
go install github.com/YOURUSERNAME/forgor@latest
```

Then:

```bash
alias ff="forgor"
```

---

## 🚀 Shell Completion

`forgor` supports shell autocompletion for bash, zsh, fish, and powershell. This gives you tab completion for commands, flags, and even smart completion for flag values like profiles and formats.

### ⚡ One-Command Setup

Just run one command and forgor will automatically set up completion for your shell:

```bash
# Auto-detect your shell and set up completion
forgor config completion

# Or specify a shell explicitly
forgor config completion zsh
forgor config completion bash
forgor config completion fish
```

**Example output:**

```bash
$ forgor config completion
🚀 Setting up zsh completion for forgor...

📋 Created backup: /Users/you/.zshrc.forgor-backup
✅ Added forgor completion to /Users/you/.zshrc
🔄 Run 'source /Users/you/.zshrc' or restart your zsh shell to enable completion
🚀 Attempting to source the file automatically...
✨ Completion should now be active in your current session!
```

This will:

- ✅ Detect your shell automatically (or use the one you specify)
- ✅ Add the necessary completion code to your shell config (`.zshrc`, `.bashrc`, etc.)
- ✅ Create a backup of your config file first
- ✅ Try to activate completion immediately in your current session

### 🧠 Smart Completions

Once set up, you'll get intelligent tab completion for:

- **`--profile`**: Your configured profiles (e.g., "openai", "anthropic", "gemini")
- **`--format`**: Valid formats ("plain", "json")
- **`--history`**: Common values (0, 1, 2, 3, 5, 10)
- **Commands**: All available forgor commands and subcommands
- **Flags**: All available flags with descriptions

---

## ⚙️ How it works

1. **Input**: You type a prompt or fix.
2. **History** (optional): `forgor` reads past commands with `-h N` (defaults to 10 max).
3. **LLM Prompting**: Sends context to an LLM (OpenAI, local model, whatever you plug in).
4. **Output**: Returns a shell command. You copy it, tweak it, or run it.

---

## 💡 Possible Commands & Ideas

- `ff turn this command into a one-liner`
- `ff add sudo if necessary to the last 2 commands -h 2`
- `ff explain this: awk '{print $2}'`
- `ff make this cross-platform -h 1`
- `ff add a dry-run flag to last command`
- `ff combine these 2 commands -h 2`
- `ff generate a find command that excludes node_modules`

Mandem thinking ahead:

- **Interactive mode**? (`ff -i`) where it asks follow-ups.
- **Explain + Suggest** mode? (`ff -e`) to get breakdown + alt options.
- Pipe in from `stdin`?:

  ```bash
  echo "rm -rf /" | ff explain this command
  ```

---

## 🧱 Roadmap (you gotta build the tings)

- [ ] CLI tool scaffolding in Go
- [ ] Shell history reader (bash/zsh/fish support)
- [ ] LLM API integration (OpenAI / local model)
- [ ] Config file for tokens/models
- [ ] Output formatting
- [ ] Interactive tweaks (`--confirm`, etc.)
- [ ] Plugins/extensions?

---

## 🔐 Privacy

If you're sending stuff to an API, watch out for sensitive info. You can bake in filters, redaction, or local-only options.

---

## 🧑‍💻 Dev Setup

```bash
git clone https://github.com/YOURUSERNAME/forgor
cd forgor
go run main.go "how do I use rsync to copy a folder"
```

---

## 🤝 Contributions

Wanna help mandem build `forgor`? Pattern a PR, raise an issue, or shout on Discord.

---

## 🕊️ License

MIT. You're free to build, remix, and shell out.

---

Let me know if you want help building the actual CLI scaffolding next fam — we'll spin this ting up faster than a drill beat at 140 BPM.
