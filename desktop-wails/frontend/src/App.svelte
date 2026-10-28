<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { DownloadStream, CancelDownload, PickFolder } from '../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

  interface Track {
    id: number
    number: number
    title: string
    status: 'waiting' | 'downloading' | 'converting' | 'done' | 'error'
    progress: number
  }

  let url = $state('')
  let output = $state('')
  let format = $state('mp3')
  let parallel = $state(3)
  let concurrency = $state(3)
  let downloading = $state(false)
  let tracks = $state<Track[]>([])
  let selectedTracks = $state<Set<number>>(new Set())
  let statusMessage = $state('Ready')
  let showSettings = $state(false)

  let totalTracks = $derived(tracks.length)
  let completedTracks = $derived(tracks.filter((t) => t.status === 'done').length)

  onMount(() => {
    EventsOn('track-discovered', (payload: { id: number; title: string }) => {
      const existing = tracks.find((t) => t.id === payload.id)
      if (!existing) {
        tracks = [
          ...tracks,
          {
            id: payload.id,
            number: tracks.length + 1,
            title: payload.title || `Track ${tracks.length + 1}`,
            status: 'downloading',
            progress: 0,
          },
        ]
        statusMessage = `Downloading... ${tracks.length} tracks found`
      }
    })

    EventsOn('track-progress', (payload: { id: number; percent: number; status: string }) => {
      tracks = tracks.map((t) => {
        if (t.id === payload.id) {
          return { ...t, status: 'downloading' as const, progress: Math.round(payload.percent) }
        }
        return t
      })
    })

    EventsOn('track-complete', (payload: { id: number; success: boolean }) => {
      tracks = tracks.map((t) => {
        if (t.id === payload.id) {
          return {
            ...t,
            status: payload.success ? ('done' as const) : ('error' as const),
            progress: payload.success ? 100 : t.progress,
          }
        }
        return t
      })
      const done = tracks.filter((t) => t.status === 'done').length
      statusMessage = `Downloading... ${done}/${totalTracks} complete`
    })

    EventsOn(
      'download-complete',
      (payload: { code: number; total: number; failed: number }) => {
        downloading = false
        const { code, total, failed } = payload
        if (code === 0 && failed === 0) {
          statusMessage = `Complete! ${total} tracks downloaded successfully.`
        } else if (failed > 0) {
          statusMessage = `Finished: ${total - failed} succeeded, ${failed} failed.`
        } else {
          statusMessage = `Finished with errors (exit code: ${code})`
        }
      }
    )
  })

  onDestroy(() => {
    EventsOff('track-discovered', 'track-progress', 'track-complete', 'download-complete')
  })

  async function pickFolder() {
    const selected = await PickFolder()
    if (selected) output = selected
  }

  function pasteUrl() {
    navigator.clipboard.readText().then((text) => {
      if (text.trim()) url = text.trim()
    })
  }

  function addUrl() {
    if (!url.trim()) return
    statusMessage = `Playlist URL set: ${url.substring(0, 50)}...`
  }

  async function startAllDownloads() {
    if (!url.trim()) {
      statusMessage = 'Error: Please provide a playlist URL.'
      return
    }
    if (!output.trim()) {
      statusMessage = 'Error: Please choose an output folder.'
      await pickFolder()
      if (!output.trim()) return
    }

    tracks = []
    selectedTracks = new Set()
    downloading = true
    statusMessage = 'Starting download...'

    try {
      await DownloadStream(url.trim(), output, format, parallel, concurrency)
      // download-complete event already updated the UI; this is just cleanup
    } catch (e) {
      if (downloading) {
        statusMessage = `Error: ${e}`
        downloading = false
      }
    }
  }

  async function cancelDownload() {
    try {
      CancelDownload()
      downloading = false
      statusMessage = 'Download cancelled.'
    } catch (e) {
      statusMessage = `Cancel failed: ${e}`
    }
  }

  function removeSelected() {
    if (selectedTracks.size === 0) return
    tracks = tracks.filter((t) => !selectedTracks.has(t.id))
    selectedTracks = new Set()
    tracks = tracks.map((t, i) => ({ ...t, number: i + 1 }))
    statusMessage = 'Selected items removed.'
  }

  function toggleSelect(id: number) {
    const newSet = new Set(selectedTracks)
    if (newSet.has(id)) newSet.delete(id)
    else newSet.add(id)
    selectedTracks = newSet
  }

  function selectAll() {
    if (selectedTracks.size === tracks.length) selectedTracks = new Set()
    else selectedTracks = new Set(tracks.map((t) => t.id))
  }

  function getStatusText(track: Track): string {
    switch (track.status) {
      case 'waiting':     return 'Waiting'
      case 'downloading': return `Downloading ${track.progress}%`
      case 'converting':  return 'Converting...'
      case 'done':        return 'Done'
      case 'error':       return 'Error'
      default:            return ''
    }
  }

  function getStatusColor(track: Track): string {
    switch (track.status) {
      case 'waiting':     return '#666'
      case 'downloading': return '#0078d4'
      case 'converting':  return '#e68a00'
      case 'done':        return '#06b025'
      case 'error':       return '#d42020'
      default:            return '#666'
    }
  }
</script>

<div class="app-window">
  <!-- Toolbar -->
  <div class="toolbar">
    <div class="toolbar-group">
      <button class="toolbar-btn" onclick={pasteUrl} title="Paste URL from clipboard">
        <span class="toolbar-icon">📋</span>
        <span class="toolbar-label">Paste URL</span>
      </button>

      <div class="toolbar-separator"></div>

      <button
        class="toolbar-btn primary"
        onclick={startAllDownloads}
        disabled={downloading || !url.trim()}
        title="Start downloading all tracks"
      >
        <span class="toolbar-icon">▶</span>
        <span class="toolbar-label">Start All</span>
      </button>

      {#if downloading}
        <button class="toolbar-btn danger" onclick={cancelDownload} title="Cancel current download">
          <span class="toolbar-icon">⏹</span>
          <span class="toolbar-label">Stop</span>
        </button>
      {/if}

      <div class="toolbar-separator"></div>

      <button
        class="toolbar-btn"
        onclick={removeSelected}
        disabled={selectedTracks.size === 0}
        title="Remove selected items"
      >
        <span class="toolbar-icon">✕</span>
        <span class="toolbar-label">Remove</span>
      </button>

      <button class="toolbar-btn" onclick={selectAll} title="Select / Deselect all">
        <span class="toolbar-icon">☑</span>
        <span class="toolbar-label">Select All</span>
      </button>
    </div>

    <div class="toolbar-group">
      <button
        class="toolbar-btn"
        onclick={() => (showSettings = !showSettings)}
        title="Settings"
      >
        <span class="toolbar-icon">⚙</span>
        <span class="toolbar-label">Options</span>
      </button>
    </div>
  </div>

  <!-- Settings Panel -->
  {#if showSettings}
    <div class="settings-panel">
      <div class="settings-row">
        <label class="settings-label">
          Concurrency (fragments per video):
          <select bind:value={concurrency} disabled={downloading}>
            {#each [1, 2, 3, 4, 5] as n}
              <option value={n}>{n}</option>
            {/each}
          </select>
        </label>
      </div>
    </div>
  {/if}

  <!-- URL Bar -->
  <div class="url-bar">
    <div class="url-label">Playlist URL:</div>
    <input
      type="text"
      class="url-input"
      bind:value={url}
      placeholder="https://youtube.com/playlist?list=PLxxxxxxx"
      disabled={downloading}
      onkeydown={(e) => e.key === 'Enter' && addUrl()}
    />
    <button class="url-add-btn" onclick={addUrl} disabled={downloading || !url.trim()}>
      Add
    </button>
  </div>

  <!-- Column Headers -->
  <div class="list-header">
    <div class="col-check">#</div>
    <div class="col-title">Title</div>
    <div class="col-format">Format</div>
    <div class="col-status">Status</div>
    <div class="col-progress">Progress</div>
  </div>

  <!-- Download Queue -->
  <div class="download-queue">
    {#if tracks.length === 0}
      <div class="empty-state">
        <div class="empty-icon">🎵</div>
        <div class="empty-text">No tracks in queue</div>
        <div class="empty-subtext">
          Paste a YouTube playlist URL above and click "Start All" to begin downloading
        </div>
      </div>
    {:else}
      {#each tracks as track, index}
        <div
          class="track-item"
          class:selected={selectedTracks.has(track.id)}
          class:even={index % 2 === 0}
          class:odd={index % 2 !== 0}
          onclick={() => toggleSelect(track.id)}
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') toggleSelect(track.id)
          }}
          role="row"
          tabindex="0"
        >
          <div class="track-row">
            <div class="col-check">
              <input
                type="checkbox"
                checked={selectedTracks.has(track.id)}
                onclick={(e) => { e.stopPropagation(); toggleSelect(track.id) }}
              />
              <span class="track-number">{track.number}</span>
            </div>
            <div class="col-title">
              <span class="track-title">{track.title}</span>
            </div>
            <div class="col-format">
              <span class="format-badge">{format.toUpperCase()}</span>
            </div>
            <div class="col-status">
              <span class="status-text" style="color: {getStatusColor(track)}">
                {getStatusText(track)}
              </span>
            </div>
            <div class="col-progress">
              <div class="progress-bar-container">
                <div
                  class="progress-bar-fill"
                  class:complete={track.status === 'done'}
                  class:error={track.status === 'error'}
                  class:converting={track.status === 'converting'}
                  style="width: {track.progress}%"
                ></div>
              </div>
              <span class="progress-text">{track.progress}%</span>
            </div>
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Status Bar -->
  <div class="status-bar">
    <div class="status-left">
      <span class="status-item">Tracks: {totalTracks}</span>
      <span class="status-divider">|</span>
      <span class="status-item">Completed: {completedTracks}</span>
      <span class="status-divider">|</span>
      <span class="status-item status-message">{statusMessage}</span>
    </div>
    <div class="status-right">
      <label class="status-control">
        Format:
        <select bind:value={format} disabled={downloading} class="status-select">
          <option value="mp3">MP3</option>
          <option value="m4a">M4A</option>
          <option value="flac">FLAC</option>
          <option value="opus">Opus</option>
          <option value="wav">WAV</option>
        </select>
      </label>
      <label class="status-control">
        Parallel:
        <select bind:value={parallel} disabled={downloading} class="status-select">
          {#each [1, 2, 3, 4, 5] as n}
            <option value={n}>{n}</option>
          {/each}
        </select>
      </label>
      <div class="status-control output-path">
        <span class="output-label" title={output}>{output || 'No folder selected'}</span>
        <button class="browse-btn" onclick={pickFolder} disabled={downloading}>📁</button>
      </div>
    </div>
  </div>
</div>

<style>
  :global(*) { box-sizing: border-box; }

  :global(body) {
    margin: 0;
    padding: 0;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    font-size: 13px;
    background: #f0f0f0;
    color: #1a1a1a;
    overflow: hidden;
    user-select: none;
    -webkit-user-select: none;
  }

  .app-window {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    background: #f0f0f0;
    border: 1px solid #999;
  }

  /* ── Toolbar ── */
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 8px;
    background: linear-gradient(to bottom, #f8f8f8, #e8e8e8);
    border-bottom: 1px solid #c0c0c0;
    min-height: 42px;
    flex-shrink: 0;
  }
  .toolbar-group { display: flex; align-items: center; gap: 2px; }
  .toolbar-btn {
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; gap: 1px; padding: 4px 10px;
    border: 1px solid transparent; border-radius: 3px;
    background: transparent; cursor: pointer;
    font-family: inherit; font-size: 11px; color: #1a1a1a;
    min-width: 56px; transition: all 0.1s;
  }
  .toolbar-btn:hover:not(:disabled) {
    background: linear-gradient(to bottom, #ffffff, #e8f0ff);
    border-color: #aecff7;
  }
  .toolbar-btn:active:not(:disabled) {
    background: linear-gradient(to bottom, #d6e8ff, #bdd7f0);
    border-color: #7eb4ea;
  }
  .toolbar-btn:disabled { opacity: 0.4; cursor: default; }
  .toolbar-btn.primary .toolbar-icon { color: #0078d4; }
  .toolbar-btn.danger .toolbar-icon  { color: #d42020; }
  .toolbar-icon  { font-size: 16px; line-height: 1; }
  .toolbar-label { font-size: 10px; color: #444; white-space: nowrap; }
  .toolbar-separator { width: 1px; height: 28px; background: #c0c0c0; margin: 0 6px; }

  /* ── Settings ── */
  .settings-panel {
    padding: 8px 12px; background: #fafafa;
    border-bottom: 1px solid #d0d0d0; flex-shrink: 0;
  }
  .settings-row { display: flex; align-items: center; gap: 16px; }
  .settings-label { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #333; }
  .settings-label select {
    padding: 2px 4px; font-size: 12px; border: 1px solid #c0c0c0;
    border-radius: 2px; background: #fff; font-family: inherit;
  }

  /* ── URL bar ── */
  .url-bar {
    display: flex; align-items: center; padding: 6px 10px;
    background: #fafafa; border-bottom: 1px solid #d0d0d0; gap: 8px; flex-shrink: 0;
  }
  .url-label { font-size: 12px; font-weight: 600; color: #333; white-space: nowrap; }
  .url-input {
    flex: 1; padding: 5px 8px; border: 1px solid #b0b0b0; border-radius: 2px;
    font-size: 12px; font-family: inherit; background: #ffffff; color: #1a1a1a;
    outline: none; transition: border-color 0.15s;
  }
  .url-input:focus { border-color: #0078d4; box-shadow: 0 0 0 1px rgba(0,120,212,0.2); }
  .url-input:disabled { background: #f0f0f0; color: #999; }
  .url-add-btn {
    padding: 5px 16px;
    background: linear-gradient(to bottom, #0084e8, #006cbe);
    color: #fff; border: 1px solid #005a9e; border-radius: 2px;
    font-size: 12px; font-weight: 600; font-family: inherit; cursor: pointer;
    transition: all 0.1s;
  }
  .url-add-btn:hover:not(:disabled) { background: linear-gradient(to bottom, #1a91eb, #0078d4); }
  .url-add-btn:disabled { opacity: 0.5; cursor: default; }

  /* ── List header ── */
  .list-header {
    display: flex; align-items: center; padding: 4px 10px;
    background: linear-gradient(to bottom, #ffffff, #f0f0f0);
    border-bottom: 1px solid #d0d0d0; font-size: 11px; font-weight: 600;
    color: #444; flex-shrink: 0;
  }
  .col-check   { width: 50px; display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
  .col-title   { flex: 1; min-width: 0; padding-right: 8px; }
  .col-format  { width: 60px; text-align: center; flex-shrink: 0; }
  .col-status  { width: 130px; flex-shrink: 0; }
  .col-progress { width: 160px; display: flex; align-items: center; gap: 6px; flex-shrink: 0; }

  /* ── Queue ── */
  .download-queue {
    flex: 1; overflow-y: auto; overflow-x: hidden;
    background: #ffffff; border-bottom: 1px solid #d0d0d0;
  }
  .download-queue::-webkit-scrollbar { width: 17px; }
  .download-queue::-webkit-scrollbar-track {
    background: #f0f0f0; border-left: 1px solid #d0d0d0;
  }
  .download-queue::-webkit-scrollbar-thumb {
    background: #c0c0c0; border: 1px solid #a0a0a0;
  }
  .download-queue::-webkit-scrollbar-thumb:hover { background: #a0a0a0; }
  .empty-state {
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; height: 100%; padding: 40px; color: #999;
  }
  .empty-icon { font-size: 48px; margin-bottom: 12px; opacity: 0.5; }
  .empty-text { font-size: 16px; font-weight: 500; color: #666; margin-bottom: 6px; }
  .empty-subtext { font-size: 12px; color: #999; text-align: center; max-width: 340px; }

  /* ── Track items ── */
  .track-item { cursor: pointer; border-bottom: 1px solid #f0f0f0; }
  .track-item.even { background: #ffffff; }
  .track-item.odd  { background: #f8f9fa; }
  .track-item:hover    { background: #e8f4fd; }
  .track-item.selected { background: #cce5ff; }
  .track-item.selected:hover { background: #b8daff; }
  .track-row { display: flex; align-items: center; padding: 8px 10px; }
  .track-number { font-size: 11px; color: #888; min-width: 20px; }
  .track-item .col-check input[type='checkbox'] { margin: 0; cursor: pointer; }
  .track-title {
    font-size: 12px; color: #1a1a1a;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block;
  }
  .format-badge {
    display: inline-block; padding: 1px 6px; background: #e8e8e8;
    border: 1px solid #c0c0c0; border-radius: 2px;
    font-size: 10px; font-weight: 600; color: #555; text-transform: uppercase;
  }
  .status-text { font-size: 11px; font-weight: 500; }

  /* ── Progress bar ── */
  .progress-bar-container {
    flex: 1; height: 14px; background: #e8e8e8;
    border: 1px solid #c0c0c0; border-radius: 1px; overflow: hidden; position: relative;
  }
  .progress-bar-fill {
    height: 100%;
    background: linear-gradient(to bottom, #2dd23b, #06b025 40%, #05a020 60%, #049a1c);
    transition: width 0.3s ease; position: relative;
  }
  .progress-bar-fill::after {
    content: ''; position: absolute; top: 0; left: 0; right: 0; height: 40%;
    background: linear-gradient(to bottom, rgba(255,255,255,0.3), transparent);
  }
  .progress-bar-fill.error {
    background: linear-gradient(to bottom, #f05050, #d42020 40%, #c01818);
  }
  .progress-bar-fill.converting {
    background: linear-gradient(to bottom, #ffbb33, #e68a00 40%, #cc7a00);
    animation: pulse-bar 1s ease-in-out infinite alternate;
  }
  @keyframes pulse-bar { from { opacity: 0.85; } to { opacity: 1; } }
  .progress-text { font-size: 10px; color: #666; min-width: 30px; text-align: right; }

  /* ── Status bar ── */
  .status-bar {
    display: flex; align-items: center; justify-content: space-between;
    padding: 4px 10px; background: #f0f0f0; border-top: 1px solid #c0c0c0;
    min-height: 30px; flex-shrink: 0; gap: 12px;
  }
  .status-left {
    display: flex; align-items: center; gap: 8px;
    font-size: 11px; color: #444; min-width: 0; flex-shrink: 1;
  }
  .status-item  { white-space: nowrap; }
  .status-message { overflow: hidden; text-overflow: ellipsis; min-width: 0; }
  .status-divider { color: #c0c0c0; }
  .status-right { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
  .status-control {
    display: flex; align-items: center; gap: 4px;
    font-size: 11px; color: #444; white-space: nowrap;
  }
  .status-select {
    padding: 1px 4px; font-size: 11px; border: 1px solid #c0c0c0;
    border-radius: 2px; background: #ffffff; font-family: inherit; color: #1a1a1a;
  }
  .status-select:disabled { background: #f0f0f0; color: #999; }
  .output-path { max-width: 200px; overflow: hidden; }
  .output-label {
    font-size: 11px; color: #555;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    max-width: 140px; display: inline-block; vertical-align: middle;
  }
  .browse-btn {
    padding: 1px 6px; border: 1px solid #c0c0c0; border-radius: 2px;
    background: linear-gradient(to bottom, #f8f8f8, #e8e8e8);
    cursor: pointer; font-size: 12px; line-height: 1; transition: all 0.1s;
  }
  .browse-btn:hover:not(:disabled) {
    background: linear-gradient(to bottom, #ffffff, #f0f0f0); border-color: #a0a0a0;
  }
  .browse-btn:disabled { opacity: 0.5; cursor: default; }
</style>
