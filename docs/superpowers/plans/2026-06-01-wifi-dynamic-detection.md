# Dynamic Wi-Fi Detection & Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Настроить динамическое определение Wi-Fi интерфейса на сервере `zwarder-server` с сохранением статического IP-адреса `192.168.0.254/24` при помощи Netplan.

**Architecture:** Мы заменим жестко заданное имя интерфейса `wlx18a6f71bd787` на логическое имя `wifi-dynamic` с использованием директивы `match: name: "wl*"`. Это позволит применить конфигурацию к любому беспроводному интерфейсу, имя которого начинается с `wl` (включая новый `wlp1s0`).

**Tech Stack:** Netplan, systemd-networkd, wpa_supplicant, Linux SSH.

---

### Task 1: Подготовка и резервное копирование

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml` (через SSH)

- [ ] **Step 1: Создать резервную копию конфигурационного файла на сервере**

  Выполнить команду для создания копии файла `/etc/netplan/01-netcfg.yaml.bak`:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.bak"
  ```
  *Expected:* Команда завершается успешно без вывода ошибок.

- [ ] **Step 2: Проверить успешное создание резервной копии**

  Выполнить:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "ls -la /etc/netplan/01-netcfg.yaml.bak"
  ```
  *Expected:* Выводится информация о созданном резервном файле.

---

### Task 2: Обновление конфигурации Netplan

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml`

- [ ] **Step 1: Подготовить обновленное содержимое файла конфигурации**

  Мы заменим жестко прописанный блок `wifis` на динамический:
  ```yaml
    wifis:
      wifi-dynamic:
        match:
          name: "wl*"
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

- [ ] **Step 2: Записать обновленный конфигурационный файл на удаленный сервер**

  Для этого мы сформируем полный файл `/etc/netplan/01-netcfg.yaml` и запишем его одной командой.
  
  Выполнить запись:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cat << 'EOF' > /etc/netplan/01-netcfg.yaml
  network:
    version: 2
    renderer: networkd
    ethernets:
      eth-en:
        match:
          name: \"en*\"
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
          name: \"eth*\"
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
      wifi-dynamic:
        match:
          name: \"wl*\"
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
          \"Trisad\":
            password: \"9657Stop@2\"
  EOF"
  ```
  *Expected:* Запись выполнена успешно.

- [ ] **Step 3: Проверить записанный файл**

  Выполнить команду для проверки структуры:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "cat /etc/netplan/01-netcfg.yaml"
  ```
  *Expected:* Вывод соответствует спецификации и содержит экранированные кавычки в нужных местах.

---

### Task 3: Проверка и применение конфигурации

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml`

- [ ] **Step 1: Запустить тестовую проверку Netplan**

  Мы используем безопасную команду `netplan try` с таймаутом, чтобы предотвратить потерю доступа при случайной сетевой ошибке:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "netplan try --timeout 10"
  ```
  *Expected:* Команда возвращает успешный синтаксический разбор конфигурации или ожидает подтверждения (поскольку мы запускаем её неинтерактивно, мы можем использовать также команду `netplan generate` для синтаксической валидации).
  Давайте сначала сгенерируем тестовую конфигурацию:
  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "netplan generate"
  ```
  *Expected:* Команда генерирует конфигурацию без ошибок синтаксиса.

- [ ] **Step 2: Применить новую конфигурацию сети**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "netplan apply"
  ```
  *Expected:* Команда завершается с кодом 0.

---

### Task 4: Верификация

**Files:**
- Modify: `root@10.8.1.4:/etc/netplan/01-netcfg.yaml`

- [ ] **Step 1: Проверить статус сетевого Wi-Fi интерфейса wlp1s0**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "ip a show wlp1s0"
  ```
  *Expected:* Статус `UP` (или `LOWER_UP`), присвоен IP-адрес `192.168.0.254/24`.

- [ ] **Step 2: Проверить состояние соединения Wi-Fi**

  ```bash
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no root@10.8.1.4 -p 6767 "iw dev wlp1s0 link"
  ```
  *Expected:* Вывод указывает на активное подключение к SSID "Trisad".

- [ ] **Step 3: Убедиться в доступности шлюза и сохранении доступа по WireGuard**

  ```bash
  ping 10.8.1.4
  ```
  *Expected:* Пинг проходит стабильно.
