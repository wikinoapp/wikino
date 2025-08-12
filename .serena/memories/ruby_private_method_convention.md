# Ruby プライベートメソッドの定義規約

## 基本方針
プライベートメソッドは必ず `private def` の形式で定義する。

## 良い例
```ruby
# typed: strict
# frozen_string_literal: true

class Example
  sig { params(value: String).returns(String) }
  def public_method(value)
    process_value(value)
  end

  # プライベートメソッドは private def で定義
  sig { params(value: String).returns(String) }
  private def process_value(value)
    value.upcase
  end
  
  sig { returns(T::Boolean) }
  private def valid?
    true
  end
end
```

## 悪い例（避けるべき）
```ruby
class Example
  def public_method(value)
    process_value(value)
  end

  private  # この形式は使わない

  def process_value(value)
    value.upcase
  end
  
  def valid?
    true
  end
end
```

## 理由
1. **明確性**: メソッド定義を見ただけでプライベートメソッドであることが分かる
2. **一貫性**: コードベース全体で統一された書き方
3. **リファクタリング**: メソッドの移動や順序変更が容易
4. **可読性**: privateキーワードとメソッド定義が離れていないため理解しやすい

## Sorbetとの併用
Sorbetの型シグネチャと併用する場合：
```ruby
sig { params(user_record: UserRecord).returns(T::Boolean) }
private def is_member?(user_record:)
  # 実装
end
```

## 注意点
- `private` を独立した行として使用しない
- 各プライベートメソッドに個別に `private def` を付ける
- プロテクテッドメソッドも同様に `protected def` を使用する