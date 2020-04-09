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
const something = "";
vector<int> test{1, 2, 3, 4, 5};
int main()
{
	for(const auto &i : test)
		cout << i << "\\n";
	cout << "Hello World!\\n";
	return 0;
}
`,
            options: {
                theme: 'monokai',
                tabSize: 4,
                lineNumbers: true,
                line: true,
                keyMap: 'sublime'
            }
        }
    }
}
</script>
