# typed: strict
# frozen_string_literal: true

# 権限チェック結果のメモ化を提供するモジュール
module Memoizable
  extend T::Sig
  extend T::Helpers

  requires_ancestor { ApplicationPolicy }

  sig { returns(T::Hash[Symbol, T.untyped]) }
  def memoized_results
    @memoized_results ||= T.let({}, T.nilable(T::Hash[Symbol, T.untyped]))
    @memoized_results.not_nil!
  end

  # メモ化ヘルパーメソッド
  # 同じ引数で同じメソッドが呼ばれた場合、キャッシュした結果を返す
  sig { params(method_name: Symbol, args: T::Hash[Symbol, T.untyped], block: T.proc.returns(T::Boolean)).returns(T::Boolean) }
  def memoize(method_name, args, &block)
    cache_key = generate_cache_key(method_name, args)

    if memoized_results.key?(cache_key)
      T.cast(memoized_results[cache_key], T::Boolean)
    else
      result = yield
      memoized_results[cache_key] = result
      result
    end
  end

  # キャッシュをクリアする
  sig { void }
  def clear_memoization!
    @memoized_results = {}
  end

  private

  # キャッシュキーを生成
  sig { params(method_name: Symbol, args: T::Hash[Symbol, T.untyped]).returns(Symbol) }
  def generate_cache_key(method_name, args)
    # argsのIDだけを使ってキーを生成（ActiveRecordオブジェクトの場合）
    serialized_args = args.map do |key, value|
      if value.respond_to?(:id)
        "#{key}:#{value.id}"
      else
        "#{key}:#{value}"
      end
    end.join("_")

    :"#{method_name}_#{serialized_args}"
  end
end

