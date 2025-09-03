# typed: strict
# frozen_string_literal: true

module Pages
  class BulkRestoringForm < ApplicationForm
    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    sig { returns(T.nilable(T::Array[Types::DatabaseId])) }
    attr_accessor :page_ids

    validates :user_record, presence: true
    validates :page_ids, presence: true
    validate :restoring_ability

    sig { returns(PageRecord::PrivateRelation) }
    private def pages
      PageRecord.where(id: page_ids)
    end

    sig { void }
    private def restoring_ability
      return if user_record.nil?

      unless user_record.not_nil!.joined_all_topics?(topic_ids: pages.pluck(:topic_id))
        errors.add(:base, :not_joined_topic_exists)
      end
    end
  end
end
