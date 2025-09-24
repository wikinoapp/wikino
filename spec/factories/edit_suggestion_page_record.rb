# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_page_record do
    space_record
    edit_suggestion_record
    page_record
    page_revision_record

    # 保存前にlatest_revision_recordの設定をスキップ
    after(:build) do |record|
      record.instance_eval {
        def skip_latest_revision_validation
          true
        end
      }
    end

    trait :new_page do
      page_record { nil }
      page_revision_record { nil }
    end
  end
end
