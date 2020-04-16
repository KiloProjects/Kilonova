<template>
    <div>
        <no-ssr placeholder="Loading...">
            <codemirror v-model="code" :options="options"></codemirror>
        </no-ssr>
    </div>
</template>
<script>
export default {
    data() {
        return {
            code: `#include <iostream>
using namespace std;

int main()
{
    cout << "Hello world!\\n";
	return 0;
}
`,
            options: {
                theme: 'monokai',
                mode: 'text/x-c++src',
                tabSize: 4,
                lineNumbers: true,
                line: true,
                keyMap: 'sublime'
            }
        }
    }
}
</script>
