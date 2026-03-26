# geo-conv

Минималистичный конвертер гео-баз из формата **v2ray/xray** (`.dat`) в формат **sing-box** (`.db`).

- `geoip.dat` → `geoip.db` (MMDB-формат sing-box)
- `geosite.dat` → `geosite.db` (бинарный формат sing-box)

## Зачем

Проект [SagerNet/v2box](https://github.com/SagerNet/v2box) перестал собираться на современных системах из-за зависимостей на gVisor, xray-core и wireguard. Эта утилита решает **только** задачу конвертации гео-данных, без лишних зависимостей.

**Ключевые отличия от v2box:**
- Нет зависимости на gVisor, wireguard, sing-tun, xray-core, sing-box (целиком)
- Используются только: protobuf (для чтения `.dat`) и mmdbwriter (для записи MMDB)
- Собирается стандартной командой `go build` без правок
- Кроссплатформенно: Windows, Linux, macOS

## Готовые бинарники

На странице [Releases](https://github.com/BLXCKBXXST/geodatamigrator/releases) доступны готовые сборки для:

| Платформа | Файл |
|---|---|
| Linux x86_64 | `geo-conv-linux-amd64` |
| Linux ARM64 | `geo-conv-linux-arm64` |
| Windows x86_64 | `geo-conv-windows-amd64.exe` |
| macOS x86_64 | `geo-conv-darwin-amd64` |
| macOS ARM64 (M1/M2) | `geo-conv-darwin-arm64` |

Скачай, сделай `chmod +x` (на Linux/macOS) и пользуйся — Go не нужен.

## Требования для сборки

- **Go 1.22+** (рекомендуется 1.22 или новее)
- Подключение к интернету (для скачивания зависимостей при первой сборке)

## Сборка

### Linux (Ubuntu)

```bash
# Установить Go (если ещё не установлен)
# https://go.dev/doc/install

# Клонировать/скопировать проект
cd geo-conv

# Скачать зависимости и собрать
go mod tidy
go build -ldflags="-s -w" -o geo-conv .

# Проверить
./geo-conv version
```

### Windows (PowerShell)

```powershell
# Установить Go: https://go.dev/dl/
# Открыть PowerShell в папке проекта

cd geo-conv

go mod tidy
go build -ldflags="-s -w" -o geo-conv.exe .

# Проверить
.\geo-conv.exe version
```

### Кросс-компиляция

```bash
# Собрать для Windows из Linux
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o geo-conv.exe .

# Собрать для Linux из Windows (в PowerShell)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o geo-conv
```

## Использование

### Конвертация GeoIP

```bash
geo-conv geoip -i geoip.dat -o geoip.db
```

### Конвертация GeoSite

```bash
geo-conv geosite -i geosite.dat -o geosite.db
```

### Параметры

| Флаг | Описание | По умолчанию |
|------|----------|--------------|
| `-i` | Путь к входному `.dat` файлу | `geoip.dat` / `geosite.dat` |
| `-o` | Путь к выходному `.db` файлу | `geoip.db` / `geosite.db` |

### Полный сценарий

```bash
# 1. Скачать исходные файлы (например, от v2fly)
wget https://github.com/v2fly/geoip/releases/latest/download/geoip.dat
wget https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat -O geosite.dat

# Или от Loyalsoldier
# wget https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
# wget https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat

# 2. Конвертировать
geo-conv geoip -i geoip.dat -o geoip.db
geo-conv geosite -i geosite.dat -o geosite.db

# 3. Скопировать результат в рабочую директорию sing-box
cp geoip.db geosite.db /etc/sing-box/
# или для Nekobox — в соответствующую папку клиента
```

## Использование с RoscomVPN

Для пользователей из России, которые используют кастомные geo-базы от [RoscomVPN](https://github.com/hydraponique):

```bash
# Скачать dat-файлы от RoscomVPN
wget https://github.com/hydraponique/roscomvpn-geoip/raw/master/release/geoip.dat
wget https://github.com/hydraponique/roscomvpn-geosite/raw/master/release/geosite.dat

# Конвертировать
geo-conv geoip -i geoip.dat -o geoip.db
geo-conv geosite -i geosite.dat -o geosite.db

# Результат: geoip.db (~0.5 MB), geosite.db (~0.1 MB)
# Скопировать в рабочую директорию sing-box / Nekobox
```

RoscomVPN использует стандартный protobuf-формат v2ray, поэтому конвертируется без проблем. В их geoip.dat содержатся записи для `ru`, `private` и других категорий, в geosite.dat — списки заблокированных доменов.

## Совместимость

### Входные форматы (`.dat`)

Поддерживаются файлы в protobuf-формате, совместимом с:
- [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core) (GeoIPList / GeoSiteList)
- [XTLS/Xray-core](https://github.com/XTLS/Xray-core) (тот же формат protobuf)
- [v2fly/geoip](https://github.com/v2fly/geoip)
- [v2fly/domain-list-community](https://github.com/v2fly/domain-list-community)
- [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)
- Любые другие источники, использующие стандартный v2ray protobuf (RoscomVPN и др.)

### Выходные форматы (`.db`)

- **geoip.db** — MaxMind DB (MMDB) с `DatabaseType: "sing-geoip"`, совместим с sing-box v1.0+
- **geosite.db** — бинарный формат sing-box (version 0), совместим с sing-box v1.0 – v1.11.x

> **Примечание:** В sing-box v1.8.0 форматы geoip.db/geosite.db помечены как deprecated в пользу rule-set (`.srs`). Однако они продолжают работать до v1.12.0. Многие клиенты (Nekobox, и др.) по-прежнему используют `.db`.

### Как переносятся теги/категории

**GeoIP:**
- Каждый `GeoIP.country_code` из `.dat` → отдельная запись в MMDB
- Код переводится в lowercase (например, `RU` → `ru`)
- Все коды сохраняются в MMDB metadata Languages для возможности перечисления

**GeoSite:**
- Каждый `GeoSite.country_code` → ключ в `.db` (в lowercase)
- Домены с атрибутами создают дополнительные ключи: `code@attribute`
  (например, `google@cn` содержит только домены Google с атрибутом `cn`)
- Типы доменов конвертируются:
  - `Plain` → `DomainKeyword`
  - `Regex` → `DomainRegex`
  - `Full` → `Domain`
  - `RootDomain` → `Domain` (если содержит `.`) + `DomainSuffix` (с префиксом `.`)

## Структура проекта

```
geo-conv/
├── main.go              # CLI и логика конвертации
├── go.mod               # Зависимости Go-модуля
├── go.sum
├── proto/
│   ├── geodata.proto    # Protobuf-определения (GeoIPList, GeoSiteList)
│   └── geodata.pb.go    # Сгенерированный Go-код (уже в репо)
└── geositedb/
    └── writer.go        # Запись geosite.db в бинарном формате sing-box
```

### Зависимости

| Пакет | Назначение |
|-------|-----------|
| `google.golang.org/protobuf` | Чтение protobuf `.dat` файлов |
| `github.com/maxmind/mmdbwriter` | Запись MMDB (формат `geoip.db`) |

Ни gVisor, ни xray-core, ни sing-box не требуются.

## Проверка результата

### Быстрая проверка через sing-box

```json
{
  "log": { "level": "info" },
  "route": {
    "geoip": { "path": "./geoip.db" },
    "geosite": { "path": "./geosite.db" },
    "rules": [
      {
        "geoip": ["ru"],
        "outbound": "direct"
      },
      {
        "geosite": ["google"],
        "outbound": "proxy"
      }
    ]
  },
  "inbounds": [],
  "outbounds": [
    { "type": "direct", "tag": "direct" },
    { "type": "block", "tag": "proxy" }
  ]
}
```

```bash
sing-box check -c config.json
# Если ошибок нет — файлы валидны
```

### Проверка через sing-box CLI

```bash
# Список кодов GeoIP
sing-box geoip list -f geoip.db

# Поиск IP
sing-box geoip lookup -f geoip.db 8.8.8.8

# Список кодов GeoSite
sing-box geosite list -f geosite.db

# Поиск домена
sing-box geosite lookup -f geosite.db google
```

## Лицензия

MIT
