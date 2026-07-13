# fortyhours

A CLI to fill out a [Productive.io](https://productive.io) timesheet and
time off without clicking through the web UI. Single static binary, no
runtime dependencies.

## Install

```sh
make build        # ./fortyhours for your current OS/arch
make dist         # dist/fortyhours-{darwin,linux}-{amd64,arm64}
```

## Configure

Productive requires an API token and organization ID on every request
([docs](https://developer.productive.io/guides/authorization)). Generate a
token in Productive under **Settings → API integrations**.

```sh
export PRODUCTIVE_API_KEY=...
export PRODUCTIVE_ORG_ID=...
```

These env vars always take precedence over the config file. Then run:

```sh
fortyhours init
```

`init` resolves your Productive person id from your email, asks which
absence events mean "sick" and "pto", and helps you pick the projects/hours
`autofill` should log by default (e.g. 7h on `dreamfi` + 1h on `internal` =
an 8h day). Everything it asks for a flag will skip that prompt, so it can
be run non-interactively:

```sh
fortyhours init --email me@example.com \
  --sick-event Sick --pto-event PTO \
  --autofill "dreamfi:7,internal:1"
```

This writes `$FORTYHOURS_CONFIG`, or `<user config dir>/fortyhours/config.yaml`
(e.g. `~/.config/fortyhours/config.yaml` on Linux) otherwise.

## Usage

```sh
fortyhours projects list
fortyhours services list --project dreamfi

fortyhours time list --from 2024-03-01 --to 2024-03-31
fortyhours time create --project dreamfi --date today --hours 7 --note "sprint work"
fortyhours time update 12345 --hours 6.5
fortyhours time delete 12345

fortyhours bookings list
fortyhours bookings create --project dreamfi --from 2024-04-01 --to 2024-04-05 --hours 8

fortyhours sick            # today
fortyhours sick 2024-03-04
fortyhours pto 2024-04-01 2024-04-05

fortyhours autofill day
fortyhours autofill week
fortyhours autofill month
fortyhours autofill 2024-04-30   # today through an explicit upper-bound date
fortyhours autofill week --dry-run
```

Every list/get command accepts `--json` for scripting.

### How autofill works

For every Monday-Friday in the range, autofill:

1. Skips the day if it already has time entries.
2. Skips the day if it's covered by an absence booking (from `sick`/`pto`).
3. Otherwise logs the configured default hours per project (from `init`, or
   `--fill "project:hours,..."` for a one-off override).
4. Warns (doesn't fail) if a day's total doesn't match the configured daily
   goal (default 8h).

The intended workflow is to schedule `autofill` (cron, launchd, Task
Scheduler) for the common case, and run `sick`/`pto`/`time create` by hand
for exceptions — autofill will then skip whatever those commands already
accounted for.

### Sick/PTO and existing time entries

Booking `sick` or `pto` over a day that already has time entries deletes
those time entries first: Productive doesn't consider tracked time and
booked absence mutually exclusive, so fortyhours enforces "one or the
other" itself. If a sick/PTO booking is later deleted directly in
Productive, fortyhours does not recreate the time entries it removed.

## Development

```sh
make test
make lint
make generate   # regenerate internal/productive/models_gen.go after
                 # updating internal/productive/spec/*.yaml (see
                 # internal/productive/spec/tools/update_spec.py)
```

See `internal/productive/generate.go` for why only resource models are
code-generated and the JSON:API client is hand-written.
