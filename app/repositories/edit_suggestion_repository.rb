# typed: strict
# frozen_string_literal: true

class EditSuggestionRepository < ApplicationRepository
  sig { params(edit_suggestion_record: EditSuggestionRecord).returns(EditSuggestion) }
  def to_model(edit_suggestion_record:)
    EditSuggestion.new(
      database_id: edit_suggestion_record.id,
      title: edit_suggestion_record.title,
      description: edit_suggestion_record.description,
      status: EditSuggestionStatus.deserialize(edit_suggestion_record.status),
      applied_at: edit_suggestion_record.applied_at,
      created_at: edit_suggestion_record.created_at,
      updated_at: edit_suggestion_record.updated_at,
      space: SpaceRepository.new.to_model(space_record: edit_suggestion_record.space_record.not_nil!),
      topic: TopicRepository.new.to_model(topic_record: edit_suggestion_record.topic_record.not_nil!),
      created_space_member: SpaceMemberRepository.new.to_model(space_member_record: edit_suggestion_record.created_space_member_record.not_nil!)
    )
  end

  sig { params(edit_suggestion_records: T::Enumerable[EditSuggestionRecord]).returns(T::Array[EditSuggestion]) }
  def to_models(edit_suggestion_records:)
    # N+1を避けるため関連データをpreload
    records = T.unsafe(edit_suggestion_records)
    if records.is_a?(ActiveRecord::Relation)
      records = records.preload(:space_record, :topic_record, :created_space_member_record)
    end

    records.map { |record| to_model(edit_suggestion_record: record) }
  end
end
