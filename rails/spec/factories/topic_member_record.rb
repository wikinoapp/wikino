# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :topic_member_record do
    space_record
    topic_record
    space_member_record
    joined_at { Time.current }
    scopes { [] }

    # no-op: 7-2 で role カラム削除後に trait 自体も削除予定。
    # 現状は topic_members.scopes は空配列で運用するため、
    # 限定スコープのテストはメンバー管理 UI 実装時に追加する。
    trait :admin do
    end

    trait :member do
    end
  end
end
