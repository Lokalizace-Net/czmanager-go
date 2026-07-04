<script lang="ts">
  import Button from './Button.svelte'
  import { Heart } from 'lucide-svelte'
</script>

<div class="welcome-wrapper">
  <div class="welcome-card">
    <div class="welcome-content">
      <div class="welcome-text">
        <h1>Vítejte v CZManager</h1>
        <p>
          Manažer českých lokalizací od skupiny Lokalizace.NET. Provoz serverů, domény a nákup her
          vyžaduje vaši podporu. Pomozte nám pokračovat v naší práci!
        </p>
      </div>
      <Button
        variant="primary"
        class="support-btn"
        onclick={() => window.open('https://lokalizace.net/support', '_blank')}
      >
        <Heart size={16} />
        Podpořit projekt
      </Button>
    </div>
  </div>
</div>

<style>
  .welcome-wrapper {
    padding: 24px 48px;
  }

  .welcome-card {
    background: linear-gradient(135deg, #1e1e1e 0%, #252525 100%);
    border: 1px solid #333;
    border-radius: 16px;
    overflow: hidden;
  }

  .welcome-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 32px;
    padding: 24px 32px;
  }

  .welcome-text {
    flex: 1;
  }

  .welcome-text h1 {
    font-size: 20px;
    font-weight: 700;
    color: white;
    margin: 0 0 8px 0;
  }

  .welcome-text p {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.6);
    margin: 0;
    line-height: 1.5;
  }
</style>
