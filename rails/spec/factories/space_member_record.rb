# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :space_member_record do
    space_record
    user_record
    active { true }
    joined_at { Time.current }
    scopes { ["space:admin"] }

    # no-op: 7-2 で role カラム削除後に trait 自体も削除予定。
    # デフォルトの scopes: ["space:admin"] で owner 相当。
    trait :owner do
    end

    # no-op: 7-2 で role カラム削除後に trait 自体も削除予定。
    # 現状は全メンバーが space:admin のため、限定スコープのテストは
    # メンバー管理 UI 実装時に追加する。
    trait :member do
    end
  end
end
