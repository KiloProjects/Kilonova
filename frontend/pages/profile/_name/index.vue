<template>
    <div>
        <Profile v-if="profil" :user="profil" />
        <ErrorCard :err="error" />
    </div>
</template>
<script>
import ErrorCard from '~/components/ErrorCard'
import Profile from '~/components/Profile'
export default {
    components: {
        ErrorCard,
        Profile
    },
    async asyncData({ $axios, params }) {
        try {
            const data = await $axios.get(`/user/getByName?name=${params.name}`)
            return {
                profil: data.data.data
            }
        } catch (err) {
            return {
                error: err.response.data
            }
        }
    },
    data() {
        return {
            profil: null,
            error: null
        }
    }
}
</script>
