# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class CreateForm < ApplicationForm
    sig { returns(T.nilable(SpaceMemberRecord)) }
    attr_accessor :space_member_record

    sig { returns(T.nilable(PageRecord)) }
    attr_accessor :page_record

    sig { returns(T.nilable(EditSuggestionRecord)) }
    attr_accessor :edit_suggestion_record

    sig { returns(T.nilable(TopicRecord)) }
    attr_accessor :topic_record

    attribute :edit_suggestion_id, :string
    attribute :page_title, :string
    attribute :page_body, :string, default: ""

    validates :space_member_record, presence: true
    validates :edit_suggestion_id, presence: true
    validates :page_title, presence: true, length: {maximum: 255}
    validate :validate_edit_suggestion_exists

    sig { void }
    private def validate_edit_suggestion_exists
      return if edit_suggestion_id.blank?
      return if space_member_record.nil?
      return if topic_record.nil?

      self.edit_suggestion_record = EditSuggestionRecord
        .open_or_draft
        .where(
          id: edit_suggestion_id,
          topic_id: topic_record.not_nil!.id,
          created_space_member_id: space_member_record.not_nil!.id
        )
        .first

      if edit_suggestion_record.nil?
        errors.add(:edit_suggestion_id, :not_found)
      end
    end
  end
end
