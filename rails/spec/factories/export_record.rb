# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export_record do
    space_record
    queued_by_record factory: :space_member_record

    trait :queued do
      after(:create) do |export_record|
        export_record.change_status!(kind: ExportStatusKind::Queued)
      end
    end

    trait :started do
      after(:create) do |export_record|
        export_record.change_status!(kind: ExportStatusKind::Started)
      end
    end

    trait :failed do
      after(:create) do |export_record|
        export_record.change_status!(kind: ExportStatusKind::Failed)
      end
    end

    trait :succeeded do
      after(:create) do |export_record|
        export_record.change_status!(kind: ExportStatusKind::Succeeded)
      end
    end
  end
end
