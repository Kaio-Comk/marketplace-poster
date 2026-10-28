# Facebook Marketplace Auto Poster

Publica anúncios no Facebook Marketplace automaticamente, no Windows, usando o seu próprio Chrome. Preenche título, preço, categoria, estado, descrição, etiquetas e localização, envia a foto, escolhe a cidade e publica.

Roda 100% local. A sessão do Facebook fica num perfil do Chrome dentro da pasta do projeto (`browser_profile\`); o programa nunca guarda a sua senha.

## Modos de uso

| Modo | Comando | Para que serve |
|---|---|---|
| Manual | `marketplace-poster.exe` | Publica os produtos da pasta `itens\` (uma subpasta por produto). |
| Automático | `marketplace-poster.exe -auto` | Publica a cota do dia a partir de uma planilha (várias cidades, variações de foto, título e descrição). |
| Daemon | `marketplace-poster.exe -daemon` | Roda sem parar, em horário comercial, e retoma de onde parou depois de queda de energia ou reinício. |

## Requisitos

- Windows 10 ou superior
- [Go 1.24+](https://go.dev/dl/) (só para compilar)
- Google Chrome
- Conta do Facebook com acesso ao Marketplace (interface em português ou inglês)

## Instalação

```
copy .env.example .env
go build -o marketplace-poster.exe .
```

Na primeira execução o Chrome abre: entre no Facebook. A sessão fica salva em `browser_profile\` e as próximas execuções já entram logadas.

## Configuração (`.env`)

| Chave | Obrigatória | Descrição |
|---|---|---|
| `PLANILHA_PATH` | nos modos `-auto` e `-daemon` | Caminho da planilha `.xlsx`. |
| `ITEMS_PATH` | não | Pasta dos produtos do modo manual. Padrão: `.\itens`. |
| `DEFAULT_CITY` | não | Cidade dos anúncios sem cidade própria. Vazio = a da conta do Facebook. |
| `CHROME_PATH` | não | Caminho do `chrome.exe`, se não for detectado. |
| `GEMINI_API_KEY` | não | Reescreve as descrições do modo manual com o Gemini. |
| `PHONE_NUMBER` | não | Telefone (só números) incluído nas descrições reescritas. Vazio = não inclui. |

O `.env` nunca vai para o git.

## Modo manual

Crie uma subpasta por produto em `itens\`, com as fotos (jpg, jpeg, png, webp, heic, heif; até 6, escolhidas ao acaso) e um `description.txt`:

```
título: Produto de exemplo
preço: 10
categoria: Celulares
estado: Usado - Em bom estado
etiquetas: exemplo, teste
cidade: Fortaleza, CE
descrição: Texto do anúncio.
Pode ter várias linhas.
```

- As chaves aceitam português ou inglês (`título`/`title`, `preço`/`price`, `categoria`/`category`, `estado`/`condition`, `descrição`/`description`, `etiquetas`/`tags`, `cidade`/`location`).
- Categoria e estado aceitam o nome em português ou inglês, com ou sem acento.
- A cidade vem, nesta ordem, de `cidade:` no arquivo, da pasta que contém o produto (`itens\Fortaleza\Produto\description.txt` publica em Fortaleza), de `-cidade` ou de `DEFAULT_CITY`.

## Modo automático (planilha)

A planilha precisa das abas **Config**, **Fotos**, **Títulos**, **Descrições**, **Cidades** e **Anúncios publicados** (a aba **Usuários** existe no modelo, mas o programa não a usa). Cada anúncio é registrado em "Anúncios publicados".

- A cota diária sobe sozinha a partir da data de início na aba Config: 3 por dia na primeira semana, 5 na segunda, 8 na terceira e 12 depois (máximo de 15).
- As cidades saem em ordem de população. Foto, título e descrição são sorteados de forma equilibrada entre as variações ativas.
- Pausa aleatória entre anúncios de 20 a 40 minutos (`-pausa-min`, `-pausa-max`).
- Se a planilha estiver aberta no Excel, o registro vai para `pendentes.csv` e entra na próxima execução. Feche o Excel antes de rodar.

```
marketplace-poster.exe -auto -simular   # só mostra o que seria publicado hoje
marketplace-poster.exe -auto            # publica a cota do dia
```

## Modo daemon (24/7)

```
marketplace-poster.exe -daemon
```

- **Horário comercial:** só publica entre `-janela-ini` e `-janela-fim` (padrão 8h às 18h, horário local). Fora disso espera.
- **Horários que mudam todo dia:** o primeiro anúncio do dia sai com atraso aleatório de até 90 minutos depois da abertura, e os demais se espalham de forma irregular até o fim do expediente. O piso entre anúncios é `-pausa-min`.
- **Retoma de onde parou:** o estado (`estado.json`) é gravado logo depois de cada publicação, antes de buscar o link e registrar na planilha. Depois de um reinício ele espera o horário que faltava e a cota do dia vem da planilha, então nada é publicado em dobro.
- **Falhas:** se o Facebook recusar a publicação ou a sessão expirar, ele espera 30 min, 1 h, 2 h, 4 h e depois 12 h antes de tentar de novo. Não derruba o programa.
- **Encerrar:** `Ctrl+C` fecha o Chrome e sai limpo.

### Subir sozinho com o Windows

1. `run-daemon.bat` mantém o programa vivo (se cair, sobe de novo em 60 s) e grava os logs em `daemon.log`.
2. Crie a tarefa do Agendador (PowerShell):

```powershell
$dir = 'C:\caminho\do\projeto'
$action = New-ScheduledTaskAction -Execute 'cmd.exe' -Argument '/c run-daemon.bat' -WorkingDirectory $dir
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -MultipleInstances IgnoreNew -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
Register-ScheduledTask -TaskName 'MarketplacePoster' -Action $action -Trigger $trigger -Settings $settings -Principal $principal
```

3. Para voltar sozinho depois de um reinício, ative o logon automático do Windows (por exemplo com o Autologon da Sysinternals) e, na BIOS, a opção de ligar após falta de energia (`Restore AC Power Loss` ou `State After G3`).

O Chrome precisa de uma sessão do Windows aberta, por isso o gatilho é "ao fazer logon" e não um serviço.

## Opções de linha de comando

| Opção | Padrão | Descrição |
|---|---|---|
| `-items` | `.\itens` | Pasta dos produtos (modo manual). |
| `-cidade` | | Cidade padrão dos anúncios sem cidade própria. |
| `-auto` | desligado | Publica a cota do dia a partir da planilha. |
| `-daemon` | desligado | Roda sem parar, retomando de `estado.json`. |
| `-simular` | desligado | Com `-auto`: mostra o que seria publicado, sem abrir o navegador. |
| `-ensaio` | desligado | Preenche o formulário até a tela final, marca os grupos sugeridos e **não clica em Publicar**. Serve para testar depois de uma mudança no Facebook. |
| `-planilha` | `PLANILHA_PATH` | Caminho da planilha. |
| `-pausa-min`, `-pausa-max` | 20, 40 | Pausa entre anúncios, em minutos (`-pausa-max` só vale no `-auto`). |
| `-janela-ini`, `-janela-fim` | 8, 18 | Com `-daemon`: horas locais em que publica. |

## Limitações

- **Preço sem centavos:** o campo de preço do Marketplace do Brasil só aceita reais inteiros. Um preço como 9,90 sai arredondado (10).
- **Telas do Facebook mudam:** o programa usa seletores da página; uma mudança no Facebook pode quebrar o preenchimento e exigir ajuste.
- **Termos da Meta:** automatizar o site pode violar as regras do Facebook e levar ao bloqueio da conta ou do Marketplace. Use conta própria e volume baixo.
- **Computador desligado não publica:** o daemon retoma quando o PC liga e a conta é aberta.

## Privacidade

Estes itens ficam só no seu computador e estão no `.gitignore`: `.env`, `browser_profile\` (sessão do Facebook), `itens\`, planilhas e fotos, `estado.json`, `daemon.log`, `pendentes.csv` e o executável. Confira com `git ls-files` antes de publicar qualquer coisa.

## Estrutura

```
main.go             opções e escolha do modo
auto.go             modo automático (planilha)
daemon.go           loop do modo daemon
daemon_aux.go       janela de horário, backoff, navegador
daemon_agenda.go    sorteio dos horários
estado.go           estado persistente (estado.json)
run-daemon.bat      mantém o daemon vivo
utils/              login, formulário do anúncio, planilha, planejador,
                    cidades, idiomas, Gemini, configuração
```

## Desenvolvimento

```
go vet ./...
go test ./...
go build -o marketplace-poster.exe .
```

## Licença

Sem licença definida: todos os direitos reservados ao autor deste repositório.

## Origem

A ideia de automatizar o formulário do Marketplace com Go e go-rod partiu do projeto público `HARON416/Facebook-Marketplace-Auto-Poster`. O código deste repositório foi reescrito do zero a partir da descrição do comportamento e da observação do formulário do Facebook; os modos automático e daemon, a planilha, o planejador de cidades e horários, a localização e os idiomas são originais deste projeto.
