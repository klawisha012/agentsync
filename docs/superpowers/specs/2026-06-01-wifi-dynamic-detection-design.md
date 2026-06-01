# Спецификация дизайна: Динамическое определение и настройка Wi-Fi адаптера через Udev и Netplan

Эта спецификация описывает переход от жестко заданной конфигурации имени Wi-Fi интерфейса к динамическому определению беспроводного PCI-E модуля на сервере `zwarder-server` при помощи udev-правила с сохранением статического IP-адреса в Netplan.

## 1. Цели
* **Динамическое определение:** Обеспечить автоматическое применение сетевых настроек к любому подключенному Wi-Fi адаптеру без необходимости вручную менять его имя в конфигурационных файлах Netplan.
* **Сохранение статического IP:** Гарантировать назначение статического IP-адреса `192.168.0.254/24` для беспроводного соединения.
* **Стабильность сети:** Сохранить работоспособность остальных сетевых интерфейсов (`enp2s0`, `wg0` WireGuard-туннеля) под управлением `systemd-networkd`.

---

## 2. Текущее состояние и проблема
На сервере `zwarder-server` (Ubuntu 24.04, ядро 6.8.0) используется сетевой стек `systemd-networkd`, управляемый через утилиту **Netplan**.

Текущий файл конфигурации `/etc/netplan/01-netcfg.yaml` содержит жесткую привязку к MAC/имени старого Wi-Fi адаптера:
```yaml
  wifis:
    wlx18a6f71bd787:
      dhcp4: false
      addresses:
        - 192.168.0.254/24
      ...
```

Сетевой бэкенд `systemd-networkd` (рендерер `networkd`) **не поддерживает** динамическое сопоставление `match` для беспроводных адаптеров в Netplan. Команда `netplan generate` возвращает ошибку:
`ERROR: wifi-dynamic: networkd backend does not support wifi with match:, only by interface name`

---

## 3. Предлагаемые изменения

### Шаг 1: Создание udev-правила для динамического переименования
We create a udev rule at `/etc/udev/rules.d/70-persistent-wifi.rules`. This rule intercepts any wireless PCI-E adapter (matched by class `0x028000`) or USB wireless adapter and renames the interface to **`wlan-default`**.

#### [NEW] `/etc/udev/rules.d/70-persistent-wifi.rules`
```udev
# Динамическое переименование любого PCI-E Wi-Fi адаптера в wlan-default
SUBSYSTEM=="net", ACTION=="add", SUBSYSTEMS=="pci", ATTRS{class}=="0x028000", NAME="wlan-default"

# Резервное правило для USB-адаптеров или других беспроводных карт
SUBSYSTEM=="net", ACTION=="add", KERNEL=="wlan*", NAME="wlan-default"
SUBSYSTEM=="net", ACTION=="add", KERNEL=="wl*", NAME="wlan-default"
```

### Шаг 2: Изменение файла `/etc/netplan/01-netcfg.yaml`
Мы настроем Netplan на работу с фиксированным интерфейсом `wlan-default`.

#### [MODIFY] `/etc/netplan/01-netcfg.yaml`
```yaml
  wifis:
    wlan-default:
      dhcp4: false
      addresses:
        - 192.168.0.254/24
      routes:
        - to: default
          via: 192.168.0.1
          metric: 200
      nameservers:
        addresses:
          - 8.8.8.8
          - 1.1.1.1
      access-points:
        "Trisad":
          password: "9657Stop@2"
```

---

## 4. План верификации и внедрения

### Этап 1: Резервное копирование
Перед внесением изменений создается резервная копия текущей конфигурации:
```bash
cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.bak
```

### Этап 2: Применение изменений
1. Записать udev-правило в `/etc/udev/rules.d/70-persistent-wifi.rules`.
2. Записать обновленный `/etc/netplan/01-netcfg.yaml`.
3. Применить udev-правила для мгновенного переименования интерфейса:
   ```bash
   udevadm control --reload-rules && udevadm trigger
   ```
4. Применить конфигурацию Netplan:
   ```bash
   netplan apply
   ```

### Этап 3: Проверка работоспособности
1. Проверить состояние сетевых интерфейсов:
   ```bash
   ip a show wlan-default
   ```
   *Ожидаемый результат:* Интерфейс `wlan-default` находится в состоянии `UP` и имеет IP-адрес `192.168.0.254/24`.
2. Проверить статус беспроводного соединения с точкой доступа "Trisad":
   ```bash
   iw dev wlan-default link
   ```
   *Ожидаемый результат:* Статус "Connected to Trisad".
3. Убедиться, что доступность по WireGuard-туннелю (`10.8.1.4`) сохраняется.
