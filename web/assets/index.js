const glob=require('glob');

console.log(glob.sync('../templ/**/*', {nodir:true}))
