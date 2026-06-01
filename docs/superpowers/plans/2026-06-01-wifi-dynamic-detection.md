# Udev & Netplan Wi-Fi Dynamic Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Настроить динамическое определение любого Wi-Fi интерфейса на сервере `zwarder-server` при помощи udev-правила, переименовывающего его в `wlan-default`, and назначить ему статический IP-адрес `192.168.0.254/24` через Netplan.

**Architecture:** Мы создадим udev-правило, которое привязывается к аппаратному PCI-классу беспроводных устройств `0x028000` (а также по маскам `wl*` и `wlan*`) и переименовывает интерфейс в `wlan-default`. В Netplan мы настроим статический блок для `wlan-default`.

**Tech Stack:** Netplan, systemd-networkd, udev, Linux SSH.

---

### Task 1: Подготовка и резервное копирование

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml` (через SSH)

- [x] **Step 1: Создать резервную копию конфигурационного файла на сервере**
- [x] **Step 2: Проверить успешное создание резервной копии**

---

### Task 2: Создание udev-правила для динамического переименования

**Files:**
- Create: `root@10.8.1.4:/etc/udev/rules.d/70-persistent-wifi.rules`

- [ ] **Step 1: Записать udev-правило на удаленный сервер**

  Выполнить запись в файл `/etc/udev/rules.d/70-persistent-wifi.rules`:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cat << 'EOF' > /etc/udev/rules.d/70-persistent-wifi.rules
  # Динамическое переименование любого PCI-E Wi-Fi адаптера в wlan-default
  SUBSYSTEM==\"net\", ACTION==\"add\", SUBSYSTEMS==\"pci\", ATTRS{class}==\"0x028000\", NAME=\"wlan-default\"

  # Резервное правило для USB-адаптеров или других беспроводных карт
  SUBSYSTEM==\"net\", ACTION==\"add\", KERNEL==\"wlan*\", NAME=\"wlan-default\"
  SUBSYSTEM==\"net\", ACTION==\"add\", KERNEL==\"wl*\", NAME=\"wlan-default\"
  EOF"
  ```
  *Expected:* Запись выполнена успешно.

- [ ] **Step 2: Проверить записанный файл правил**

  Выполнить:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cat /etc/udev/rules.d/70-persistent-wifi.rules"
  ```
  *Expected:* Вывод соответствует спецификации udev-правила.

---

### Task 3: Обновление конфигурации Netplan

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml`

- [ ] **Step 1: Записать обновленный конфигурационный файл Netplan на удаленный сервер**

  Мы настроим статический IP для интерфейса `wlan-default`.
  
  Выполнить запись:
  ```bash
  @'
  network:
    version: 2
    renderer: networkd
    ethernets:
      eth-en:
        match:
          name: "en*"
        dhcp4: false
        addresses:
          - 192.168.0.254/24
        routes:
          - to: default
            via: 192.168.0.1
            metric: 100
        nameservers:
          addresses:
            - 8.8.8.8
            - 1.1.1.1
      eth-legacy:
        match:
          name: "eth*"
        dhcp4: false
        addresses:
          - 192.168.0.254/24
        routes:
          - to: default
            via: 192.168.0.1
            metric: 101
        nameservers:
          addresses:
            - 8.8.8.8
            - 1.1.1.1
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
  '@ | ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cat > /etc/netplan/01-netcfg.yaml"
  ```
  *Expected:* Запись выполнена успешно.

- [ ] **Step 2: Проверить записанный файл**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cat /etc/netplan/01-netcfg.yaml"
  ```
  *Expected:* Файл содержит правильную структуру с фиксированным `wlan-default`.

---

### Task 4: Применение udev-правил и Netplan

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml`, `root@10.8.1.4:/etc/udev/rules.d/70-persistent-wifi.rules`

- [ ] **Step 1: Перезагрузить и применить правила udev**

  Это заставит ядро переименовать текущий интерфейс `wlp1s0` в `wlan-default`:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "udevadm control --reload-rules && udevadm trigger"
  ```
  *Expected:* Команда завершается без ошибок.

- [ ] **Step 2: Проверить, что интерфейс успешно переименован**

  Выполнить:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "ip link show wlan-default"
  ```
  *Expected:* Выводится информация об интерфейсе `wlan-default` (он может быть в состоянии DOWN).

- [ ] **Step 3: Проверить синтаксис Netplan**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "netplan generate"
  ```
  *Expected:* Команда завершается без ошибок (код 0, без вывода).

- [ ] **Step 4: Применить конфигурацию Netplan**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "netplan apply"
  ```
  *Expected:* Команда завершается успешно.

---

### Task 5: Верификация работы сети

- [ ] **Step 1: Проверить статус сетевого интерфейса wlan-default**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "ip a show wlan-default"
  ```
  *Expected:* Статус `UP` (или `LOWER_UP`), назначен IP `192.168.0.254/24`.

- [ ] **Step 2: Проверить беспроводное соединение**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "iw dev wlan-default link"
  ```
  *Expected:* Соединение с "Trisad" активно.

- [ ] **Step 3: Убедиться в доступности шлюза и стабильности WireGuard**

  ```bash
  ping 10.8.1.4
  ```
  *Expected:* Пинг стабильно возвращает ответы.
