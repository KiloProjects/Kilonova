export const getters = {
    isAuthed(state) {
        return state.auth.loggedIn
    },

    loggedInUser(state) {
        return state.auth.user
    }
}
