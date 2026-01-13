# Windsurf TUI

> **Explore bancos PostgreSQL com a sensação de uma planilha retrô e o poder do terminal.**

## ✨ Destaques

- **Navegação hierárquica completa**: servidores → bancos → esquemas → tabelas → dados.
- **Pane Data estilo planilha**: rolagem horizontal/vertical, seleção de células e atalhos de teclado old school.
- **CRUD preparado**: atalhos para inserir, atualizar e excluir registros via `PostgresTreeLoader`.
- **UI responsiva**: os quatro painéis se ajustam automaticamente ao tamanho do terminal.
- **Conexões seguras**: `connections.json` é ignorado pelo Git para você manter credenciais fora do repositório.

## 🛠️ Pré-requisitos

- Go 1.18+ (`go env GOPATH` configurado)
- PostgreSQL acessível via rede

## 🚀 Instalação

```bash
# Clonar e entrar na pasta
 git clone <seu-fork-ou-repo>
 cd windsurf-tui

# Compilar binário
 go build -o windsurf-tui
```

## 💻 Uso rápido

1. Execute `./windsurf-tui`.
2. Configure uma conexão no diálogo inicial (as credenciais ficam em `connections.json`).
3. Explore bancos com as setas; `Enter` em uma tabela carrega os dados no painel inferior.
4. Use PageUp/PageDown/Home/End para percorrer grandes datasets.
5. CRUD: `Enter` abre edição da célula, `Ctrl+N` insere linha, `Ctrl+D` remove.

## 📦 Estrutura principal

- `main.go`: loop do Bubble Tea, mensagens e ciclo de vida da UI.
- `pane_model.go`: estado e seleção de cada painel.
- `pane_renderer.go`: rendering com Lipgloss, inclusive a planilha.
- `pane_navigator.go`: roteamento de teclas e drill-down.
- `postgres_tree_loader.go`: consultas e operações nos bancos PostgreSQL.

## ☁️ Publicar no GitHub

1. Faça login no GitHub e crie um repositório vazio (sem README).
2. No projeto local:
   ```bash
   git init
   git add .
   git commit -m "feat: primeira versao do windsuf-tui"
   git branch -M main
   git remote add origin git@github.com:<usuario>/<repo>.git
   git push -u origin main
   ```
3. Habilite GitHub Actions ou workflows extras conforme desejar.

## 🧭 Roadmap (curto prazo)

- Integração completa dos atalhos CRUD com `PostgresTreeLoader`.
- Highlight visual para células editadas/pendentes.
- Exportação CSV a partir do painel de dados.

Ficou com alguma ideia ou encontrou um bug? Abra uma issue ou mande um PR. Bora navegar bancos com estilo! 🇧🇷
