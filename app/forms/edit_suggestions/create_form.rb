# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class CreateForm < ApplicationForm
    sig { returns(T.nilable(SpaceMemberRecord)) }
    attr_accessor :space_member_record

    sig { returns(T.nilable(PageRecord)) }
    attr_accessor :page_record

    sig { returns(T.nilable(TopicRecord)) }
    attr_accessor :topic_record

    attribute :title, :string
    attribute :description, :string, default: ""
    attribute :page_title, :string
    attribute :page_body, :string, default: ""
    attribute :existing_edit_suggestion_id, :string

    validates :space_member_record, presence: true
    validates :topic_record, presence: true
    validates :title, presence: true, length: {maximum: 2}
    validates :page_title, presence: true, length: {maximum: 255}

    validate :validate_edit_suggestion_selection

    sig { returns(T::Boolean) }
    def create_new_edit_suggestion?
      existing_edit_suggestion_id.blank?
    end

    sig { returns(T.nilable(EditSuggestionRecord)) }
    def existing_edit_suggestion
      return if existing_edit_suggestion_id.blank?
      return if topic_record.nil? || space_member_record.nil?

      # 自分が作成した下書き/オープンの編集提案のみ選択可能
      EditSuggestionRecord
        .open_or_draft
        .where(
          topic_id: topic_record.not_nil!.id,
          created_space_member_id: space_member_record.not_nil!.id
        )
        .find_by(id: existing_edit_suggestion_id)
    end

    sig { void }
    private def validate_edit_suggestion_selection
      return if create_new_edit_suggestion?
      return if existing_edit_suggestion.present?

      errors.add(:existing_edit_suggestion_id, :invalid)
    end
  end
end
