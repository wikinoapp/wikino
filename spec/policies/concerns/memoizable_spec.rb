# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe Memoizable do
  # テスト用のポリシークラス
  class TestPolicy < ApplicationPolicy
    include Memoizable

    attr_reader :counter

    def initialize(user_record:)
      super
      @counter = 0
    end

    def expensive_operation(value:)
      memoize(:expensive_operation, {value:}) do
        @counter += 1
        value > 5
      end
    end

    def another_operation(x:, y:)
      memoize(:another_operation, {x:, y:}) do
        @counter += 1
        x + y > 10
      end
    end
  end

  it "同じ引数で呼ばれた場合、結果をキャッシュすること" do
    user_record = FactoryBot.create(:user_record)
    policy = TestPolicy.new(user_record:)

    # 最初の呼び出し
    result1 = policy.expensive_operation(value: 10)
    expect(result1).to eq(true)
    expect(policy.counter).to eq(1)

    # 2回目の呼び出し（キャッシュから返される）
    result2 = policy.expensive_operation(value: 10)
    expect(result2).to eq(true)
    expect(policy.counter).to eq(1) # カウンターは増えない

    # 異なる引数での呼び出し
    result3 = policy.expensive_operation(value: 3)
    expect(result3).to eq(false)
    expect(policy.counter).to eq(2) # カウンターが増える
  end

  it "複数の引数を持つメソッドで動作すること" do
    user_record = FactoryBot.create(:user_record)
    policy = TestPolicy.new(user_record:)

    # 最初の呼び出し
    result1 = policy.another_operation(x: 7, y: 6)
    expect(result1).to eq(true)
    expect(policy.counter).to eq(1)

    # 同じ引数での呼び出し（キャッシュから返される）
    result2 = policy.another_operation(x: 7, y: 6)
    expect(result2).to eq(true)
    expect(policy.counter).to eq(1)

    # 引数の順番が違っても同じ引数として扱われる
    result3 = policy.another_operation(y: 6, x: 7)
    expect(result3).to eq(true)
    expect(policy.counter).to eq(1)

    # 異なる引数での呼び出し
    result4 = policy.another_operation(x: 2, y: 3)
    expect(result4).to eq(false)
    expect(policy.counter).to eq(2)
  end

  it "clear_memoization!でキャッシュをクリアできること" do
    user_record = FactoryBot.create(:user_record)
    policy = TestPolicy.new(user_record:)

    # 最初の呼び出し
    policy.expensive_operation(value: 10)
    expect(policy.counter).to eq(1)

    # キャッシュから返される
    policy.expensive_operation(value: 10)
    expect(policy.counter).to eq(1)

    # キャッシュをクリア
    policy.clear_memoization!

    # 再度計算される
    policy.expensive_operation(value: 10)
    expect(policy.counter).to eq(2)
  end

  it "ActiveRecordオブジェクトをキャッシュキーに使用できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    policy = TestPolicy.new(user_record:)

    # ActiveRecordオブジェクトのIDをキャッシュキーに使用
    counter = 0
    policy.memoize(:with_record, {space_record:}) do
      counter += 1
      true
    end
    expect(counter).to eq(1)

    # 同じレコードで呼び出すとキャッシュから返される
    policy.memoize(:with_record, {space_record:}) do
      counter += 1
      true
    end
    expect(counter).to eq(1) # カウンターは増えない
  end
end

