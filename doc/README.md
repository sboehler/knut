# knut example journal

- Update prices from Yahoo!:

  ```
  knut fetch prices.yaml
  ```

- Create a generic balance:
    ```
    knut balance example.org
    ```

- Create a monthly balance sheet in CHF:
    ```
    knut balance example.org -v CHF -c0,"(Income|Expenses|Equity)" --monthly --to 2020-05-01
    ```

- Create an income statement in CHF (signs according to accounting conventions):
    ```
    knut balance example.org -vCHF -c0,"(Assets|Liabilities)" --monthly --to 2020-05-01 --diff
    ```
