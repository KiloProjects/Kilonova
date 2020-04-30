<template>
    <div>
        <h2>Titlu</h2>
        <b-input-group prepend="Titlu: ">
            <b-input v-model="data.title" type="text"></b-input>
        </b-input-group>
        <br />
        <b-button @click.prevent="updateTitle"> Actualizare Titlu </b-button>
        <hr />
        <h2>Descriere</h2>
        <editor v-model="data.description" lang="markdown" />
        <h4>Preview</h4>
        <hr />
        <markdown :value="data.description"></markdown>
        <hr />
        <b-button @click.prevent="updateDescription">
            Actualizare Descriere
        </b-button>
        <hr />
        <ErrorCard v-model="error" />
        <ErrorCard v-model="response" status="primary" />
        <br />
        <br />
        <br />
        <br />
        <br />
        <br />
        <br />
        <br />
        <hr />
        <h2>Preview Problema:</h2>
        <Problem :problem="data"></Problem>
    </div>
</template>
<script>
import ErrorCard from '~/components/ErrorCard'
import Markdown from '~/components/Markdown'
import Editor from '~/components/Editor'
import Problem from '~/components/Problem'
export default {
    components: {
        ErrorCard,
        Markdown,
        Editor,
        Problem
    },
    async asyncData({ params, $axios }) {
        try {
            const data = await $axios.get('/problem/getByID', {
                params: { id: params.id }
            })
            return {
                data: data.data.data
            }
        } catch (err) {
            return {
                error: err.response.data.data
            }
        }
    },
    data() {
        return {
            error: null,
            response: null,
            data: null
        }
    },
    methods: {
        async updateTitle() {
            try {
                const data = await this.$axios({
                    method: 'POST',
                    url: `/problem/update/${this.$route.params.id}/title`,
                    params: { title: this.data.title }
                })
                this.response = data.data.data
            } catch (err) {
                this.error = err.response.data.data
            }
        },
        async updateDescription() {
            try {
                const data = await this.$axios({
                    method: 'POST',
                    url: `/problem/update/${this.$route.params.id}/description`,
                    params: { description: this.data.description }
                })
                this.response = data.data.data
            } catch (err) {
                this.error = err.response.data.data
            }
        }
    }
}
</script>
