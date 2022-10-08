import { atom, selector, useRecoilState, useRecoilValue } from "recoil";
import Button from "@mui/material/Button";
import { KnutServiceClient } from "../../proto/ServiceServiceClientPb";
import { HelloRequest } from "../../proto/service_pb";

export function Counter() {
  const [counter, setCounter] = useRecoilState(counterState);
  const g = useRecoilValue(greeting);

  const increment = () => {
    setCounter(counter + 1);
  };

  const decrement = () => {
    setCounter(counter - 1);
  };

  return (
    <div>
      <p>{g}</p>
      <Button variant="contained" onClick={increment}>
        Increment
      </Button>
      <p>{counter}</p>
      <Button variant="contained" onClick={decrement}>
        Decrement
      </Button>
    </div>
  );
}

const counterState = atom({
  key: "counterState", // unique ID (with respect to other atoms/selectors)
  default: 0, // default value (aka initial value)
});

const knutService = new KnutServiceClient("");

const name = atom({
  key: "name",
  default: "foobar",
});

const greeting = selector({
  key: "greeting",
  get: async ({ get }) => {
    const req = new HelloRequest().setName(get(name));
    const res = await knutService.hello(req, {})
    return res.getGreeting();
  },
});
