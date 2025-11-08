# UniQUE API

## search

users の例

| クエリ                                                                           | 説明                                            |
| -------------------------------------------------------------------------------- | ----------------------------------------------- |
| `/users/search?filter=is_enable:true`                                            | 有効ユーザーだけ                                |
| `/users/search?filter=joined_before:2024-01-01`                                  | 2024 年 1 月 1 日より前に参加した人             |
| `/users/search?filter=created_after:2024-01-01 12:00:00`                         | 2024 年 1 月 1 日正午以降に作成されたユーザー   |
| `/users/search?filter=is_enable:true,is_suspended:false,joined_after:2023-01-01` | 有効で停止していなくて、2023 年以降に参加した人 |
| `/users/search?q=yui&filter=is_enable:true`                                      | 名前やメールに「yui」を含み、有効なユーザーだけ |
