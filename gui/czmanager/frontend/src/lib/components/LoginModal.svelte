<script lang="ts">
  import { X, User, Lock, Loader2, AlertCircle, Crown, Heart } from 'lucide-svelte'
  import { authStore } from '../stores/auth.svelte'
  import { favoritesStore } from '../stores/favorites.svelte'

  let {
    open = false,
    onClose
  }: {
    open?: boolean
    onClose?: () => void
  } = $props()

  let username = $state('')
  let password = $state('')
  let usernameInput = $state<HTMLInputElement | undefined>(undefined)
  let passwordInput = $state<HTMLInputElement | undefined>(undefined)
  let submitBtn = $state<HTMLButtonElement | undefined>(undefined)
  let closeBtn = $state<HTMLButtonElement | undefined>(undefined)

  let isLoading = $derived($authStore.isLoading)
  let error = $derived($authStore.error)

  // Focus na username input když se modal otevře
  $effect(() => {
    if (open && usernameInput) {
      setTimeout(() => usernameInput?.focus(), 100)
    }
  })

  async function handleSubmit(e: Event) {
    e.preventDefault()

    if (!username.trim() || !password) {
      return
    }

    const success = await authStore.login(username.trim(), password)

    if (success) {
      username = ''
      password = ''
      // Načti oblíbené z API po přihlášení
      favoritesStore.fetchFromApi()
      onClose?.()
    }
  }

  // Keyboard navigace pro šipky
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose?.()
      return
    }

    const target = e.target as HTMLElement
    const isUsernameInput = target === usernameInput
    const isPasswordInput = target === passwordInput
    const isSubmitBtn = target === submitBtn
    const isCloseBtn = target === closeBtn

    // Šipka dolů
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (isCloseBtn) {
        usernameInput?.focus()
      } else if (isUsernameInput) {
        passwordInput?.focus()
      } else if (isPasswordInput) {
        submitBtn?.focus()
      }
      return
    }

    // Šipka nahoru
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (isSubmitBtn) {
        passwordInput?.focus()
      } else if (isPasswordInput) {
        usernameInput?.focus()
      } else if (isUsernameInput) {
        closeBtn?.focus()
      }
      return
    }

    // Enter v inputech -> submit
    if (e.key === 'Enter' && (isUsernameInput || isPasswordInput)) {
      e.preventDefault()
      if (username.trim() && password) {
        handleSubmit(e)
      } else if (isUsernameInput) {
        passwordInput?.focus()
      }
      return
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onClose?.()
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="modal-backdrop"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
  >
    <div class="modal-container">
      <!-- Header -->
      <div class="modal-header">
        <h2>Přihlášení</h2>
        <button bind:this={closeBtn} class="close-btn" onclick={onClose}>
          <X size={20} />
        </button>
      </div>

      <!-- Content -->
      <div class="modal-content">
        <p class="login-info">
          Přihlaš se svým účtem z <a href="https://lokalizace.net" target="_blank">lokalizace.net</a>
        </p>

        {#if error}
          <div class="error-box">
            <AlertCircle size={18} />
            <span>{error}</span>
          </div>
        {/if}

        <form onsubmit={handleSubmit}>
          <div class="form-group">
            <label for="username">
              <User size={16} />
              Uživatelské jméno
            </label>
            <input
              bind:this={usernameInput}
              id="username"
              type="text"
              bind:value={username}
              placeholder="Zadej uživatelské jméno"
              disabled={isLoading}
              autocomplete="username"
            />
          </div>

          <div class="form-group">
            <label for="password">
              <Lock size={16} />
              Heslo
            </label>
            <input
              bind:this={passwordInput}
              id="password"
              type="password"
              bind:value={password}
              placeholder="Zadej heslo"
              disabled={isLoading}
              autocomplete="current-password"
            />
          </div>

          <button
            bind:this={submitBtn}
            type="submit"
            class="submit-btn"
            disabled={isLoading || !username.trim() || !password}
          >
            {#if isLoading}
              <Loader2 size={16} class="spinning" />
              Přihlašování...
            {:else}
              Přihlásit se
            {/if}
          </button>
        </form>

        <!-- Subscription info -->
        <div class="subscription-info">
          <div class="tier-card supporter">
            <Heart size={18} />
            <div>
              <strong>Supporter</strong>
              <span>Základní výhody a podpora projektu</span>
            </div>
          </div>
          <div class="tier-card vip">
            <Crown size={18} />
            <div>
              <strong>VIP</strong>
              <span>Auto-vyhledávání her, žádné reklamy</span>
            </div>
          </div>
        </div>

        <p class="register-hint">
          Nemáš účet? <a href="https://lokalizace.net/register" target="_blank">Zaregistruj se</a>
        </p>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 20px;
  }

  .modal-container {
    background: #1a1a1a;
    border-radius: 16px;
    width: 100%;
    max-width: 420px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .modal-header h2 {
    font-size: 18px;
    font-weight: 600;
    color: white;
    margin: 0;
  }

  .close-btn {
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.5);
    cursor: pointer;
    padding: 4px;
    border-radius: 6px;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .close-btn:focus {
    outline: none;
    background: rgba(255, 255, 255, 0.1);
    color: white;
    box-shadow: 0 0 0 2px #f97316;
  }

  .modal-content {
    padding: 24px;
  }

  .login-info {
    color: rgba(255, 255, 255, 0.6);
    font-size: 14px;
    margin: 0 0 20px 0;
    text-align: center;
  }

  .login-info a {
    color: #f97316;
    text-decoration: none;
  }

  .login-info a:hover {
    text-decoration: underline;
  }

  .error-box {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: 10px;
    color: #ef4444;
    font-size: 14px;
    margin-bottom: 20px;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .form-group label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.7);
  }

  .form-group input {
    padding: 12px 16px;
    background: rgba(255, 255, 255, 0.05);
    border: 2px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    font-size: 15px;
    color: white;
    outline: none;
    transition: all 0.2s;
  }

  .form-group input::placeholder {
    color: rgba(255, 255, 255, 0.3);
  }

  .form-group input:focus {
    border-color: #f97316;
    background: rgba(255, 255, 255, 0.08);
  }

  .form-group input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .submit-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 12px 24px;
    background: #f97316;
    border: 2px solid transparent;
    border-radius: 10px;
    font-size: 15px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
    outline: none;
  }

  .submit-btn:hover:not(:disabled) {
    background: #ea580c;
  }

  .submit-btn:focus {
    border-color: white;
    box-shadow: 0 0 0 3px rgba(249, 115, 22, 0.5);
  }

  .submit-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.spinning) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .subscription-info {
    margin-top: 24px;
    padding-top: 24px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .tier-card {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .tier-card.supporter {
    color: #f472b6;
  }

  .tier-card.vip {
    color: #fbbf24;
  }

  .tier-card div {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .tier-card strong {
    font-size: 14px;
    font-weight: 600;
  }

  .tier-card span {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.5);
  }

  .register-hint {
    text-align: center;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
    margin: 20px 0 0 0;
  }

  .register-hint a {
    color: #f97316;
    text-decoration: none;
  }

  .register-hint a:hover {
    text-decoration: underline;
  }
</style>
