# Rubio-IA

Un acuerdo de trabajo corto para tu agente de código, en unos 470 tokens.

Rubio-IA escribe un bloque pequeño en las instrucciones globales de Claude Code y Cursor: ser directo, verificar antes de afirmar, dimensionar el trabajo, no tocar secretos y usar [noah](https://github.com/dicapower/noah) para la memoria si está instalado. No tiene personaje ni tono propio. Para cambios grandes o de riesgo añade una skill, `rubio-plan`: planificar, aprobar, construir, verificar.

Go, solo biblioteca estándar. Sin red, sin telemetría, sin cgo. Apache-2.0, DICA POWER OÜ. [English](README.md)

## Instalación

```sh
curl -fsSL https://raw.githubusercontent.com/dicapower/rubio-ia/main/install.sh | sh
```

Homebrew:

```sh
brew tap dicapower/rubio-ia https://github.com/dicapower/rubio-ia
brew install dicapower/rubio-ia/rubio
```

Scoop (Windows):

```sh
scoop bucket add rubio https://github.com/dicapower/rubio-ia
scoop install rubio
```

O descarga un archivo de las releases; las sumas están en `SHA256SUMS`.

## Uso

```sh
rubio setup --lang es-ES      # auto (por defecto), es-ES o en
rubio doctor                  # qué hay instalado y qué compite con ello
rubio setup --dry-run         # muestra los cambios sin hacerlos
rubio remove                  # quita Rubio-IA
rubio restore --list          # cada cambio tiene copia de seguridad previa
rubio restore [ID]            # deshace el último cambio, o el que indiques
```

Reinicia tu agente después de `setup` para que recargue las instrucciones.

## Qué escribe

| Dónde | Qué |
|---|---|
| `~/.claude/CLAUDE.md` | el núcleo, entre marcas `<!-- rubio-ia:core -->`; tu texto no se toca |
| `~/.cursor/rules/rubio-ia.mdc` | el mismo núcleo como regla siempre activa |
| `~/.claude/skills/rubio-plan/`, `~/.cursor/skills/rubio-plan/` | la skill de planificación (omítela con `--no-skills`) |
| `~/.rubio-ia/backups/` | los ficheros que `rubio` tocó, los últimos diez cambios |

Una skill con el mismo nombre que hayas escrito tú nunca se sobrescribe ni se elimina.

## rubio-plan

Lo pequeño se hace sin más. Lo mediano recibe un plan de cinco líneas como máximo. Lo grande o de riesgo sigue explorar, proponer (aprobarás antes de que se construya nada), especificar y tareas, aplicar, verificar. Los cambios de alto riesgo pasan por una revisión independiente con lista de hallazgos y una sola ronda de corrección como máximo. Las notas van a noah si está disponible; si no, a `docs/plans/<cambio>.md`.

## Funciona con noah

[noah](https://github.com/dicapower/noah) es la memoria; Rubio-IA le dice al agente cómo trabajar y cuándo usarla. El núcleo solo nombra herramientas de noah (`mem_context`, `mem_search`, `mem_get`, `mem_save`, `mem_forget`) y un test lo garantiza. Sin noah esa sección se ignora y los planes van a `docs/plans/<cambio>.md`. Rubio-IA nunca toca la base de datos de noah, y noah nunca edita las instrucciones de tu agente.

## Si vienes de Gentle-AI

```sh
rubio doctor
rubio migrate gentle-ai            # muestra el plan, no cambia nada
rubio migrate gentle-ai --yes      # lo aplica, con copia de seguridad
```

Quita los bloques de Gentle-AI de `CLAUDE.md`, desactiva el estilo de salida Gentleman y borra su regla de Cursor. Con `--skills` elimina también las skills, agentes y comandos `sdd-*`; con `--plugins` desactiva el plugin de Engram. Tus otras skills se conservan. `~/.gentle-ai` no se toca; si contiene copias antiguas (pueden ocupar gigas e incluir historial de chats), `doctor` te dice el tamaño y decides tú.

## Límites

- Dos agentes por ahora: Claude Code y Cursor.
- No puede obligar a un agente a obedecer. Hace que las instrucciones sean lo bastante cortas para que se lean.
- La memoria necesita [noah](https://github.com/dicapower/noah); sin él esa sección se ignora.

## Desarrollo

```sh
go vet ./... && go test ./...
```

Un test mantiene el núcleo por debajo de 1.000 tokens y comprueba que solo nombra herramientas de noah que existen.
