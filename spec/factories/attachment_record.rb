# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :attachment_record do
    space_record
    attached_space_member_record { association :space_member_record, space_record: }
    association :active_storage_attachment_record, factory: :active_storage_attachment
    attached_at { Time.current }
    processing_status { AttachmentProcessingStatus::Completed.serialize }
    metadata { {} }

    trait :pending do
      processing_status { AttachmentProcessingStatus::Pending.serialize }
    end

    trait :processing do
      processing_status { AttachmentProcessingStatus::Processing.serialize }
    end

    trait :failed do
      processing_status { AttachmentProcessingStatus::Failed.serialize }
    end
  end
end