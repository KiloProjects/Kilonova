<template>
    <Profile :user="this.$auth.user" />
</template>
<script>
import Profile from '~/components/Profile'
export default {
    middleware: 'auth',
    components: {
        Profile
    }
}
</script>
