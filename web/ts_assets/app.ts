import {defaultClient} from './requests';

console.log(defaultClient.doRequest({url: '/', method: 'GET'}));
