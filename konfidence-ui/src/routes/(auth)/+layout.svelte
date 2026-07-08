<script lang="ts">
    let { children } = $props();
</script>

<main class="auth-shell">
    <div class="auth-card-slot">
        {@render children()}
    </div>
</main>

<style>
    .auth-shell {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 2rem 1rem;
        box-sizing: border-box;
    }

    .auth-card-slot {
        width: min(28rem, 100%);
    }
</style>
