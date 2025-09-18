# typed: strict
# frozen_string_literal: true

class EditSuggestionRepository
  extend T::Sig

  sig { params(edit_suggestion_record: EditSuggestionRecord).returns(EditSuggestion) }
  def self.to_model(edit_suggestion_record:)
    EditSuggestion.new(
      id: edit_suggestion_record.id,
      space_id: edit_suggestion_record.space_id,
      topic_id: edit_suggestion_record.topic_id,
      created_user_id: edit_suggestion_record.created_user_id,
      title: edit_suggestion_record.title,
      description: edit_suggestion_record.description,
      status: edit_suggestion_record.status,
      applied_at: edit_suggestion_record.applied_at,
      created_at: edit_suggestion_record.created_at,
      updated_at: edit_suggestion_record.updated_at
    )
  end

  sig { params(edit_suggestion_records: T::Enumerable[EditSuggestionRecord]).returns(T::Array[EditSuggestion]) }
  def self.to_models(edit_suggestion_records:)
    edit_suggestion_records.map { |record| to_model(edit_suggestion_record: record) }
  end
end
