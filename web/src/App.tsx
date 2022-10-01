import { Counter } from "./features/counter/Counter";
import { KnutServiceClient } from "./proto/ServiceServiceClientPb";
import { HelloRequest } from "./proto/service_pb";

// gRPC hello world
const knutService = new KnutServiceClient("http://localhost:7777");
const req = new HelloRequest().setName("Foobar");
knutService.hello(req, {}, function (err, response) {
  if (err) {
    console.log(err);
    return;
  } else {
    console.log(response.getGreeting());
  }
});

function App() {
  return <Counter />;
}
export default App;
