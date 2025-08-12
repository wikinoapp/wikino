# プロジェクト固有のルール（一般的なRuby/Railsと異なる点）

## 特に注意すべき独自ルール

### 1. プライベートメソッドの定義
**一般的なRuby:**
```ruby
private
def method_name
end
```

**このプロジェクト:**
```ruby
private def method_name
end
```
**理由**: メソッド定義を見ただけでプライベートであることが分かるようにするため

### 2. ActiveRecordのクエリメソッド
**一般的なRails:**
```ruby
Model.includes(:association)  # Railsが自動で最適化
```

**このプロジェクト:**
```ruby
Model.preload(:association)   # 明示的に別クエリ
Model.eager_load(:association) # 明示的にJOIN
# includesは使用禁止
```
**理由**: クエリの挙動を明確にし、予期しない動作を防ぐため

### 3. nilチェック
**一般的なSorbet:**
```ruby
T.must(variable)
```

**このプロジェクト:**
```ruby
variable.not_nil!
```
**理由**: プロジェクトで定義したヘルパーメソッドを統一的に使用

### 4. RSpecの構造
**一般的なRSpec:**
```ruby
context "when xxx" do
  let(:user) { create(:user) }
  it "does something" do
    # ...
  end
end
```

**このプロジェクト:**
```ruby
it "xxxのとき、somethingすること" do
  user = FactoryBot.create(:user)
  # ...
end
```
**理由**: テストの可読性向上とスコープの明確化

### 5. IDの生成
**一般的なRails:**
```ruby
t.bigint :id, primary_key: true
```

**このプロジェクト:**
```ruby
t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
```
**理由**: ULIDを使用してソート可能なUUIDを生成

## これらのルールを忘れずに適用するために
1. 新しいファイルを作成する前に必ず確認
2. 既存のコードを参考にする際も、これらのルールを優先
3. 一般的な慣習と異なることを常に意識する