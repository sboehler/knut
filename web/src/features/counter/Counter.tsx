import { atom, useRecoilState } from "recoil";
import Button from '@mui/material/Button';

export function Counter() {
  const [counter, setCounter] = useRecoilState(counterState);

  const increment = () => {
    setCounter(counter + 1);
  };

  const decrement = () => {
    setCounter(counter - 1);
  };

  return (
    <div>
      <Button variant="contained" onClick={increment}>Increment</Button>
      <p>{counter}</p>
      <Button variant="contained" onClick={decrement}>Decrement</Button>
    </div>
  );
}

const counterState = atom({
  key: "counterState", // unique ID (with respect to other atoms/selectors)
  default: 0, // default value (aka initial value)
});
