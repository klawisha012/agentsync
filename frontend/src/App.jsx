import { createSignal, onMount, For, Show, createMemo } from 'solid-js'

function App() {
  // Навигация
  const [activeTab, setActiveTab] = createSignal("dashboard")
  
  // Состояние данных
  const [agents, setAgents] = createSignal([])
  const [components, setComponents] = createSignal([])
  const [secrets, setSecrets] = createSignal({})
  const [requiredSecrets, setRequiredSecrets] = createSignal([])
  const [selectedCompName, setSelectedCompName] = createSignal("")
  const [activeCategory, setActiveCategory] = createSignal("MCP")
  const [activeAgents, setActiveAgents] = createSignal([])
  const [showDropdown, setShowDropdown] = createSignal(false)
  const [bundles, setBundles] = createSignal([])
  const [selectedBundleId, setSelectedBundleId] = createSignal("")
  const [selectedBundleTab, setSelectedBundleTab] = createSignal("MCP")
  
  // Секреты
  const [newSecretName, setNewSecretName] = createSignal("")
  const [newSecretVal, setNewSecretVal] = createSignal("")
  const [editingSecretName, setEditingSecretName] = createSignal("")
  const [editingSecretVal, setEditingSecretVal] = createSignal("")
  const [showSecretValue, setShowSecretValue] = createSignal(false)
  
  // Реестр (Install Hub)
  const [searchQuery, setSearchQuery] = createSignal("")
  const [searchResults, setSearchResults] = createSignal([])
  const [loadingSearch, setLoadingSearch] = createSignal(false)
  
  // Логи
  const [logs, setLogs] = createSignal([
    { time: new Date().toLocaleTimeString(), text: "Система AgentSync Web GUI инициализирована.", type: "success" }
  ])

  // Добавление лога
  const addLog = (text, type = "info") => {
    setLogs((prev) => [
      { time: new Date().toLocaleTimeString(), text, type },
      ...prev
    ])
  }

  // Загрузка данных с сервера
  const fetchStatus = async () => {
    try {
      const res = await fetch("/api/status")
      if (res.ok) {
        const data = await res.json()
        setAgents(data)
      }
    } catch (e) {
      addLog("Ошибка загрузки статуса агентов: " + e.message, "error")
    }
  }

  const fetchComponents = async () => {
    try {
      const res = await fetch("/api/components")
      if (res.ok) {
        const data = await res.json()
        setComponents(data)
      }
    } catch (e) {
      addLog("Ошибка загрузки компонентов: " + e.message, "error")
    }
  }

  const fetchSecrets = async () => {
    try {
      const res = await fetch("/api/secrets")
      if (res.ok) {
        const data = await res.json()
        setSecrets(data.values || {})
        setRequiredSecrets(data.required || [])
      }
    } catch (e) {
      addLog("Ошибка загрузки секретов: " + e.message, "error")
    }
  }

  const allSecretsList = createMemo(() => {
    const list = []
    const secValues = secrets()
    const reqSecs = requiredSecrets()
    
    reqSecs.forEach(req => {
      const exists = secValues[req.name] !== undefined
      list.push({
        name: req.name,
        value: exists ? secValues[req.name] : "",
        exists: exists,
        required: true,
        usedBy: req.usedBy.join(", ")
      })
    })
    
    Object.keys(secValues).forEach(name => {
      const alreadyAdded = list.some(item => item.name === name)
      if (!alreadyAdded) {
        list.push({
          name: name,
          value: secValues[name],
          exists: true,
          required: false,
          usedBy: "Создан вручную"
        })
      }
    })
    
    return list.sort((a, b) => a.name.localeCompare(b.name))
  })

  const fetchManifest = async () => {
    try {
      const res = await fetch("/api/manifest")
      if (res.ok) {
        const data = await res.json()
        setActiveAgents(data.active_agents || [])
        if (data.active_bundle) {
          setSelectedBundleId(data.active_bundle)
        }
      }
    } catch (e) {
      addLog("Ошибка загрузки манифеста: " + e.message, "error")
    }
  }

  const handleToggleActiveAgent = async (agentId, active) => {
    let newAgents = [...activeAgents()]
    if (active) {
      if (!newAgents.includes(agentId)) {
        newAgents.push(agentId)
      }
    } else {
      newAgents = newAgents.filter(a => a !== agentId)
    }

    try {
      const res = await fetch("/api/manifest", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          active_agents: newAgents,
          active_bundle: selectedBundleId()
        })
      })
      if (res.ok) {
        setActiveAgents(newAgents)
        addLog(`Обновлены цели синхронизации: ${newAgents.join(", ")}`, "success")
      } else {
        addLog("Не удалось сохранить цели синхронизации", "error")
      }
    } catch (e) {
      addLog("Ошибка обновления целей: " + e.message, "error")
    }
  }

  const handleSaveActiveBundle = async (bundleId) => {
    try {
      const res = await fetch("/api/manifest", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          active_agents: activeAgents(),
          active_bundle: bundleId
        })
      })
      if (res.ok) {
        setSelectedBundleId(bundleId)
        addLog(`Активная конфигурация изменена на: ${bundleId}`, "success")
      } else {
        addLog("Не удалось сохранить активную конфигурацию", "error")
      }
    } catch (e) {
      addLog("Ошибка изменения конфигурации: " + e.message, "error")
    }
  }

  const fetchBundles = async () => {
    try {
      const res = await fetch("/api/bundles")
      if (res.ok) {
        const data = await res.json()
        setBundles(data || [])
        if (data && data.length > 0 && !selectedBundleId()) {
          setSelectedBundleId(data[0].id)
        }
      }
    } catch (e) {
      addLog("Ошибка загрузки бандлов: " + e.message, "error")
    }
  }

  const selectedBundle = createMemo(() => {
    return bundles().find(b => b.id === selectedBundleId()) || null
  })

  const handleCreateBundle = async () => {
    const name = prompt("Введите имя нового бандла:")
    if (!name) return
    const desc = prompt("Введите описание бандла:")
    const id = name.toLowerCase().replace(/[^a-z0-9]/g, "-")

    const newBundle = {
      id,
      name,
      description: desc || "Без описания",
      components: []
    }

    try {
      const res = await fetch("/api/bundles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(newBundle)
      })
      if (res.ok) {
        addLog(`Бандл ${name} успешно создан!`, "success")
        fetchBundles()
        setSelectedBundleId(id)
      }
    } catch (e) {
      addLog("Ошибка создания бандла: " + e.message, "error")
    }
  }

  const handleDeleteBundle = async (id) => {
    if (!confirm("Вы действительно хотите удалить этот бандл?")) return
    try {
      const res = await fetch("/api/bundles", {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id })
      })
      if (res.ok) {
        addLog("Бандл успешно удален", "success")
        setSelectedBundleId("")
        fetchBundles()
      }
    } catch (e) {
      addLog("Ошибка удаления бандла: " + e.message, "error")
    }
  }

  const handleAddItemToBundle = async (comp) => {
    const bundle = selectedBundle()
    if (!bundle) return

    if (bundle.components.some(c => c.name === comp.Name && c.type === comp.Type)) {
      addLog("Компонент уже добавлен в этот бандл", "warning")
      return
    }

    const updatedComponents = [...bundle.components, { name: comp.Name, type: comp.Type }]
    const updatedBundle = { ...bundle, components: updatedComponents }

    try {
      const res = await fetch("/api/bundles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(updatedBundle)
      })
      if (res.ok) {
        addLog(`Компонент ${comp.Name} добавлен в бандл ${bundle.name}`, "success")
        fetchBundles()
      }
    } catch (e) {
      addLog("Ошибка добавления в бандл: " + e.message, "error")
    }
  }

  const handleRemoveItemFromBundle = async (itemName, itemType) => {
    const bundle = selectedBundle()
    if (!bundle) return

    const updatedComponents = bundle.components.filter(c => !(c.name === itemName && c.type === itemType))
    const updatedBundle = { ...bundle, components: updatedComponents }

    try {
      const res = await fetch("/api/bundles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(updatedBundle)
      })
      if (res.ok) {
        addLog(`Компонент ${itemName} удален из бандла ${bundle.name}`, "success")
        fetchBundles()
      }
    } catch (e) {
      addLog("Ошибка удаления из бандла: " + e.message, "error")
    }
  }

  const handleSyncBundle = async (id) => {
    addLog(`Запуск синхронизации компонентов бандла...`, "info")
    try {
      const res = await fetch("/api/bundles/sync", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id })
      })
      if (res.ok) {
        const data = await res.json()
        addLog(data.msg, "success")
        fetchComponents()
      }
    } catch (e) {
      addLog("Ошибка синхронизации бандла: " + e.message, "error")
    }
  }

  const handleShareBundle = async (id) => {
    addLog("Инициализация P2P-шеринга бандла...", "info")
    try {
      const res = await fetch("/api/bundles/share", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id })
      })
      if (res.ok) {
        const data = await res.json()
        addLog(`${data.msg} Использовать PIN-код: ${data.pin}`, "success")
      }
    } catch (e) {
      addLog("Ошибка запуска шеринга: " + e.message, "error")
    }
  }

  onMount(() => {
    fetchStatus()
    fetchComponents()
    fetchSecrets()
    fetchManifest()
    fetchBundles()
  })

  // Выбранный компонент
  const selectedComponent = createMemo(() => {
    return components().find(c => c.Name === selectedCompName()) || null
  })

  // Фильтрованные компоненты по категории
  const filteredComponents = createMemo(() => {
    return components().filter(c => c.Type === activeCategory())
  })

  // Действия: Smart Merge Deploy
  const handleDeploy = async () => {
    addLog(`Запуск Smart Merge деплоя для конфигурации "${selectedBundleId()}"...`, "info")
    try {
      const res = await fetch("/api/deploy", { 
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ active_bundle: selectedBundleId() })
      })
      if (res.ok) {
        addLog("Слияние и деплой успешно запущены! Настройки передаются активным агентам.", "success")
        fetchStatus()
      } else {
        addLog("Не удалось запустить слияние.", "error")
      }
    } catch (e) {
      addLog("Ошибка деплоя: " + e.message, "error")
    }
  }

  // Действие: Обновление целей синхронизации
  const handleToggleTarget = async (comp, agentName, active) => {
    let newTargets = [...comp.Targets]
    if (active) {
      if (!newTargets.includes(agentName)) {
        newTargets.push(agentName)
      }
    } else {
      newTargets = newTargets.filter(t => t !== agentName)
    }

    try {
      const res = await fetch("/api/components/update-targets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: comp.Name,
          type: comp.Type,
          targets: newTargets
        })
      })
      if (res.ok) {
        addLog(`Обновлены цели для ${comp.Name}: ${newTargets.join(", ")}`, "success")
        // Обновляем локальный стейт
        setComponents(prev => prev.map(c => {
          if (c.Name === comp.Name && c.Type === comp.Type) {
            return { ...c, Targets: newTargets }
          }
          return c
        }))
      } else {
        addLog(`Не удалось обновить цели для ${comp.Name}`, "error")
      }
    } catch (e) {
      addLog("Ошибка сети при обновлении целей: " + e.message, "error")
    }
  }

  // Действие: Синхронизация источника
  const handleSyncSource = async (comp, toGlobal) => {
    addLog(`Копирование компонента ${comp.Name} в ${toGlobal ? 'глобальный каталог' : 'локальный репозиторий (CWD)'}...`, "info")
    try {
      const res = await fetch("/api/components/sync-source", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: comp.Name,
          type: comp.Type,
          toGlobal
        })
      })
      if (res.ok) {
        addLog(`Синхронизация источника для ${comp.Name} успешно завершена!`, "success")
        fetchComponents() // перезагружаем список, чтобы обновились статусы наличия файлов
      } else {
        addLog("Не удалось синхронизировать файл.", "error")
      }
    } catch (e) {
      addLog("Ошибка синхронизации: " + e.message, "error")
    }
  }

  // Действия с секретами
  const handleSaveNewSecret = async (e) => {
    e.preventDefault()
    const name = newSecretName().trim()
    const val = newSecretVal()
    if (!name) return

    try {
      const res = await fetch("/api/secrets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, value: val })
      })
      if (res.ok) {
        addLog(`Секрет ${name} успешно сохранен!`, "success")
        setNewSecretName("")
        setNewSecretVal("")
        fetchSecrets()
        fetchComponents()
      }
    } catch (e) {
      addLog("Ошибка сохранения секрета: " + e.message, "error")
    }
  }

  const handleUpdateSecret = async (e) => {
    e.preventDefault()
    const name = editingSecretName()
    const val = editingSecretVal()

    try {
      const res = await fetch("/api/secrets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, value: val })
      })
      if (res.ok) {
        addLog(`Значение секрета ${name} обновлено!`, "success")
        setEditingSecretName("")
        setEditingSecretVal("")
        fetchSecrets()
        fetchComponents()
      }
    } catch (e) {
      addLog("Ошибка изменения секрета: " + e.message, "error")
    }
  }

  const handleDeleteSecret = async (name) => {
    if (!confirm(`Вы действительно хотите удалить секрет ${name}?`)) return
    try {
      const res = await fetch("/api/secrets", {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name })
      })
      if (res.ok) {
        addLog(`Секрет ${name} успешно удален!`, "success")
        fetchSecrets()
        fetchComponents()
      }
    } catch (e) {
      addLog("Ошибка удаления секрета: " + e.message, "error")
    }
  }

  // Действия: Поиск в хабе реестра
  const handleSearchRegistry = async (e) => {
    e.preventDefault()
    setLoadingSearch(true)
    addLog(`Запрос на поиск "${searchQuery()}" отправлен к хабу компонентов...`, "info")
    try {
      const res = await fetch(`/api/registry/search?q=${encodeURIComponent(searchQuery())}&type=MCP`)
      if (res.ok) {
        const data = await res.json()
        setSearchResults(data || [])
        addLog(`Найдено компонентов в хабе: ${data ? data.length : 0}`, "success")
      }
    } catch (e) {
      addLog("Ошибка поиска: " + e.message, "error")
    } finally {
      setLoadingSearch(false)
    }
  }

  // Действие: Установка из Хаба
  const handleInstallRegistry = async (item) => {
    addLog(`Загрузка и установка компонента ${item.name} локально в проект...`, "info")
    try {
      const res = await fetch("/api/registry/install", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(item)
      })
      if (res.ok) {
        addLog(`Компонент ${item.name} успешно установлен в текущую директорию!`, "success")
        fetchComponents()
      } else {
        addLog("Не удалось установить компонент.", "error")
      }
    } catch (e) {
      addLog("Ошибка установки: " + e.message, "error")
    }
  }

  const availableAgents = [
    { id: 'claude-code', name: 'Claude Code' },
    { id: 'claude-desktop', name: 'Claude Desktop' },
    { id: 'antigravity', name: 'Antigravity IDE' }
  ]

  const selectedAgentsText = createMemo(() => {
    const active = activeAgents()
    if (active.length === 0) return "Выберите цели"
    if (active.length === availableAgents.length) return "Все агенты"
    return `Цели: ${active.length}`
  })

  return (
    <div class="app-container">
      <header>
        <div class="logo-section">
          <h1>⚡ AGENTSYNC WEB GUI ⚡</h1>
          <p>Интерактивная панель управления и синхронизации AI-агентов</p>
        </div>
        <div class="header-actions">
          <div class="agent-select-container">
            <span class="select-label">Цели:</span>
            <div class="agent-dropdown">
              <button class="dropdown-trigger" onClick={() => setShowDropdown(!showDropdown())}>
                {selectedAgentsText()} ▾
              </button>
              <Show when={showDropdown()}>
                <div class="dropdown-menu">
                  <For each={availableAgents}>
                    {(agent) => (
                      <label class="dropdown-item">
                        <input 
                          type="checkbox" 
                          checked={activeAgents().includes(agent.id)}
                          onChange={(e) => handleToggleActiveAgent(agent.id, e.target.checked)}
                        />
                        <span>{agent.name}</span>
                      </label>
                    )}
                  </For>
                </div>
              </Show>
            </div>
          </div>

          <div class="agent-select-container">
            <span class="select-label">Конфигурация:</span>
            <div class="agent-dropdown">
              <select 
                class="dropdown-trigger"
                style="background: rgba(15, 18, 28, 0.6); border: 1px solid var(--panel-border); color: var(--cyan); padding: 8px 16px; font-size: 13px; border-radius: 8px; font-weight: 600; cursor: pointer; outline: none;"
                value={selectedBundleId()} 
                onChange={(e) => {
                  handleSaveActiveBundle(e.target.value);
                  setSelectedBundleTab("MCP");
                }}
              >
                <For each={bundles()}>
                  {(b) => <option value={b.id} style="background-color: #08090c; color: #fff;">{b.name}</option>}
                </For>
              </select>
            </div>
            <button class="primary" style="padding: 8px 12px; font-size: 13px; min-width: auto; height: 35px; align-items: center; justify-content: center; display: inline-flex;" onClick={handleCreateBundle}>➕</button>
          </div>

          <button class="primary" onClick={handleDeploy}>
            🚀 Smart Merge Deploy
          </button>
        </div>
      </header>

      <nav class="tabs-nav">
        <button class={`tab-btn ${activeTab() === 'dashboard' ? 'active' : ''}`} onClick={() => setActiveTab("dashboard")}>
          🎛 Dashboard
        </button>
        <button class={`tab-btn ${activeTab() === 'manager' ? 'active' : ''}`} onClick={() => setActiveTab("manager")}>
          💼 Agent Manager
        </button>
        <button class={`tab-btn ${activeTab() === 'secrets' ? 'active' : ''}`} onClick={() => setActiveTab("secrets")}>
          🔐 Secrets
        </button>
        <button class={`tab-btn ${activeTab() === 'hub' ? 'active' : ''}`} onClick={() => setActiveTab("hub")}>
          ☁ Install Hub
        </button>
      </nav>

      {/* Содержимое вкладок */}
      <main class="tab-content">
        
        {/* Вкладка 1: DASHBOARD */}
        <Show when={activeTab() === 'dashboard'}>
          <div class="grid-dashboard">
            <div class="panel">
              <h2>📟 Обнаруженные AI-агенты</h2>
              <For each={agents()}>
                {(agent) => (
                  <div class="agent-card">
                    <div class="agent-info">
                      <h3>{agent.name}</h3>
                      <p>{agent.config_paths[0]}</p>
                    </div>
                    <span class={`badge ${agent.detected ? 'detected' : 'not-found'}`}>
                      {agent.detected ? 'обнаружен' : 'не найден'}
                    </span>
                  </div>
                )}
              </For>
            </div>
            
            <div class="panel">
              <h2>📋 Логи событий и операций</h2>
              <div class="log-console">
                <For each={logs()}>
                  {(log) => (
                    <div class="log-entry">
                      <span class="log-time">[{log.time}]</span>
                      <span class={`log-${log.type}`}>{log.text}</span>
                    </div>
                  )}
                </For>
              </div>
            </div>
          </div>
        </Show>

        {/* Вкладка 2: AGENT MANAGER (Configs) */}
        <Show when={activeTab() === 'manager'}>
          <div class="manager-layout">
            
            {/* Панель 1: Категории (Левая) */}
            <div class="panel categories-panel">
              <h2>📁 Категории</h2>
              <div class="subcategories">
                <For each={["MCP", "Rule", "Skill", "Workflow", "Hook"]}>
                  {(cat) => (
                    <button 
                      class={`sub-btn ${activeCategory() === cat ? 'active' : ''}`}
                      onClick={() => {
                        setActiveCategory(cat)
                        if (filteredComponents().length > 0) {
                          setSelectedCompName(filteredComponents()[0].Name)
                        } else {
                          setSelectedCompName("")
                        }
                      }}
                    >
                      {cat === 'MCP' ? '🔌 MCP Серверы' : 
                       cat === 'Rule' ? '📜 Правила' : 
                       cat === 'Skill' ? '🧠 Навыки' : 
                       cat === 'Workflow' ? '🔄 Процессы' : '🪝 Хуки'}
                    </button>
                  )}
                </For>
              </div>
            </div>

            {/* Панель 2: Список компонентов (Центральная) */}
            <div class="panel components-panel">
              <h2>📦 Компоненты</h2>
              <div class="component-list">
                <For each={filteredComponents()} fallback={<p style="color: var(--text-secondary); text-align: center; padding: 20px;">Нет компонентов</p>}>
                  {(comp) => {
                    const isAdded = () => selectedBundle() ? selectedBundle().components.some(i => i.name === comp.Name && i.type === comp.Type) : false;
                    return (
                      <div 
                        class={`comp-card ${selectedCompName() === comp.Name ? 'focused' : ''} ${isAdded() ? 'active' : ''}`}
                        onClick={() => setSelectedCompName(comp.Name)}
                      >
                        <div class="comp-info">
                          <h4>{comp.Name}</h4>
                          <p>{comp.Description || "Нет описания"}</p>
                        </div>
                        <button 
                          class="as-btn" 
                          style={`padding: 4px 8px; font-size: 11px; border-color: ${isAdded() ? '#ff5555' : 'var(--cyan)'}; color: ${isAdded() ? '#ff5555' : 'var(--cyan)'};`}
                          onClick={(e) => {
                            e.stopPropagation();
                            if (isAdded()) {
                              handleRemoveItemFromBundle(comp.Name, comp.Type);
                            } else {
                              handleAddItemToBundle(comp);
                            }
                          }}
                        >
                          {isAdded() ? 'Убрать' : 'Добавить'}
                        </button>
                      </div>
                    )
                  }}
                </For>
              </div>
            </div>

            {/* Панель 3: Детали и Конфигурация (Правая) */}
            <div class="panel details-panel">
              <Show when={selectedBundle()} fallback={<p style="color: var(--text-secondary); padding: 10px;">Выберите или создайте бандл для настройки.</p>}>
                <div class="bundle-details-header" style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid rgba(255,255,255,0.08); padding-bottom: 12px; margin-bottom: 15px;">
                  <div class="bundle-details-info">
                    <h3 style="font-size: 18px; margin: 0 0 4px 0; color: var(--cyan);">{selectedBundle().name}</h3>
                    <p style="font-size: 12px; margin: 0; color: var(--text-secondary);">{selectedBundle().description || "Описание не задано"}</p>
                  </div>
                  <button class="danger" style="padding: 6px 10px; font-size: 12px;" onClick={() => handleDeleteBundle(selectedBundle().id)}>
                    🗑
                  </button>
                </div>

                {/* Горизонтальные табы категорий */}
                <h4 style="font-size: 12px; text-transform: uppercase; color: var(--cyan); margin-top: 20px; letter-spacing: 0.5px;">
                  📦 Содержимое конфигурации:
                </h4>

                <div class="bundle-tabs">
                  <For each={[
                    { id: "MCP", label: "🔌 MCP" },
                    { id: "Rule", label: "📜 Правила" },
                    { id: "Skill", label: "🧠 Навыки" },
                    { id: "Workflow", label: "🔄 Процессы" },
                    { id: "Hook", label: "🪝 Хуки" }
                  ]}>
                    {(tab) => {
                      const count = () => selectedBundle() ? selectedBundle().components.filter(c => c.type === tab.id).length : 0;
                      const isActive = () => selectedBundleTab() === tab.id;
                      return (
                        <button 
                          class={`bundle-tab-btn ${isActive() ? 'active' : ''}`}
                          onClick={() => setSelectedBundleTab(tab.id)}
                        >
                          {tab.label} ({count()})
                        </button>
                      )
                    }}
                  </For>
                </div>

                {/* Список содержимого активной вкладки */}
                <div class="bundle-content-list" style="max-height: 200px; overflow-y: auto;">
                  <For 
                    each={selectedBundle().components.filter(c => c.type === selectedBundleTab())} 
                    fallback={<p style="color: var(--text-secondary); font-size: 12px; padding: 20px 0; text-align: center;">В этой категории пока пусто.</p>}
                  >
                    {(item) => (
                      <div class="bundle-content-item">
                        <strong style="font-size: 13px; font-weight: 600;">{item.name}</strong>
                        <button class="danger" style="padding: 4px 10px; font-size: 11px;" onClick={() => handleRemoveItemFromBundle(item.name, item.type)}>
                          Убрать
                        </button>
                      </div>
                    )}
                  </For>
                </div>

              </Show>
            </div>

          </div>
        </Show>

        <Show when={activeTab() === 'secrets'}>
          <div class="grid-dashboard">
            <div class="panel">
              <h2>🔐 Секреты и токены авторизации</h2>
              <div class="component-list" style="max-height: 450px;">
                <For each={allSecretsList()}>
                  {(item) => (
                    <div class="agent-card" style={`margin-bottom: 12px; display: flex; flex-direction: column; align-items: stretch; gap: 10px; border-color: ${!item.exists ? 'var(--purple)' : 'rgba(255, 255, 255, 0.05)'}; background: ${!item.exists ? 'rgba(189, 147, 249, 0.03)' : 'rgba(255, 255, 255, 0.02)'};`}>
                      <div style="display: flex; justify-content: space-between; align-items: center;">
                        <div>
                          <h3 style={`color: ${!item.exists ? 'var(--purple)' : 'var(--cyan)'}; font-weight: 700;`}>
                            {item.name}
                          </h3>
                          <div style="margin-top: 4px; display: flex; gap: 8px; align-items: center;">
                            <Show when={!item.exists} fallback={
                              <span class="badge detected" style="font-size: 10px; padding: 2px 8px;">
                                {item.required ? `используется в ${item.usedBy}` : "Создан вручную"}
                              </span>
                            }>
                              <span class="badge not-found" style="font-size: 10px; padding: 2px 8px; color: var(--purple); border-color: rgba(189, 147, 249, 0.2); background: rgba(189, 147, 249, 0.1);">
                                ⚠️ Требуется для {item.usedBy}
                              </span>
                            </Show>
                          </div>
                        </div>

                        <div>
                          <Show when={item.exists} fallback={
                            <button class="primary" style="padding: 4px 10px; font-size: 11px;" onClick={() => { setNewSecretName(item.name); addLog(`Подготовка создания секрета для ${item.name}`, "info"); }}>
                              ➕ Создать
                            </button>
                          }>
                            <button style="padding: 4px 8px; font-size: 11px; margin-right: 8px;" onClick={() => { setEditingSecretName(item.name); setEditingSecretVal(item.value); }}>
                              ✏ Изменить
                            </button>
                            <button class="danger" style="padding: 4px 8px; font-size: 11px;" onClick={() => handleDeleteSecret(item.name)}>
                              🗑 Удалить
                            </button>
                          </Show>
                        </div>
                      </div>

                      <Show when={item.exists}>
                        <Show when={editingSecretName() === item.name} fallback={
                          <p style="font-family: var(--font-mono); font-size: 13px; word-break: break-all; color: var(--text-secondary);">
                            {showSecretValue() ? item.value : "••••••••••••••••••••••••••••"}
                          </p>
                        }>
                          <form onSubmit={handleUpdateSecret} style="margin-top: 10px; display: flex; gap: 10px;">
                            <input 
                              type={showSecretValue() ? "text" : "password"} 
                              value={editingSecretVal()} 
                              onInput={(e) => setEditingSecretVal(e.target.value)} 
                              placeholder="Новое значение"
                              style="flex: 1; padding: 6px 12px;"
                            />
                            <button type="submit" class="primary" style="padding: 6px 12px; font-size: 12px;">
                              Сохранить
                            </button>
                            <button onClick={() => setEditingSecretName("")} style="padding: 6px 12px; font-size: 12px;">
                              Отмена
                            </button>
                          </form>
                        </Show>
                      </Show>
                    </div>
                  )}
                </For>
              </div>

              <div style="margin-top: 15px;">
                <button onClick={() => setShowSecretValue(!showSecretValue())}>
                  {showSecretValue() ? "🙈 Скрыть значения" : "👁 Показать значения"}
                </button>
              </div>
            </div>

            <div class="panel">
              <h2>➕ Добавить новый секрет</h2>
              <form onSubmit={handleSaveNewSecret}>
                <div class="secret-form-group">
                  <label>ИМЯ СЕКРЕТА (КЛЮЧ)</label>
                  <input 
                    type="text" 
                    placeholder="Например, GITHUB_TOKEN" 
                    value={newSecretName()} 
                    onInput={(e) => setNewSecretName(e.target.value)} 
                    required 
                  />
                </div>
                <div class="secret-form-group">
                  <label>ЗНАЧЕНИЕ СЕКРЕТА</label>
                  <input 
                    type={showSecretValue() ? "text" : "password"} 
                    placeholder="Введите значение секрета" 
                    value={newSecretVal()} 
                    onInput={(e) => setNewSecretVal(e.target.value)} 
                    required 
                  />
                </div>
                <button type="submit" class="primary" style="width: 100%; justify-content: center; margin-top: 10px;">
                  💾 Создать секрет
                </button>
              </form>
            </div>
          </div>
        </Show>

        {/* Вкладка 4: INSTALL HUB */}
        <Show when={activeTab() === 'hub'}>
          <div class="panel">
            <h2>☁ Глобальный хаб плагинов и компонентов</h2>
            <p style="color: var(--text-secondary); margin-bottom: 20px;">
              Поиск и мгновенная установка MCP-серверов и компонентов из каталогов GitHub и Smithery.
            </p>

            <form onSubmit={handleSearchRegistry} class="search-container">
              <input 
                type="text" 
                placeholder="Введите имя плагина или тему поиска (например: sqlite, postgres)..." 
                value={searchQuery()} 
                onInput={(e) => setSearchQuery(e.target.value)} 
              />
              <button type="submit" class="primary">
                🔍 Искать в Хабе
              </button>
            </form>

            <Show when={loadingSearch()}>
              <div style="text-align: center; padding: 40px; color: var(--cyan);">
                ⏳ Выполняется запрос к реестру GitHub API... Пожалуйста, подождите.
              </div>
            </Show>

            <div class="registry-grid">
              <For each={searchResults()}>
                {(item) => (
                  <div class="registry-card">
                    <div>
                      <h3>🔌 {item.name}</h3>
                      <p>{item.description}</p>
                    </div>
                    <div class="registry-footer">
                      <span class="registry-url">{item.sourceUrl}</span>
                      <button class="primary" style="padding: 6px 12px; font-size: 12px;" onClick={() => handleInstallRegistry(item)}>
                        📥 Установить
                      </button>
                    </div>
                  </div>
                )}
              </For>
            </div>
          </div>
        </Show>

      </main>
    </div>
  )
}

export default App
