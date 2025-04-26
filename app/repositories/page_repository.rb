# typed: strict
# frozen_string_literal: true

class PageRepository < ApplicationRepository
  sig { params(page_record: PageRecord, can_update: T.nilable(T::Boolean)).returns(Page) }
  def to_model(page_record:, can_update: nil)
    Page.new(
      database_id: page_record.id,
      number: page_record.number,
      title: page_record.title,
      body: page_record.body,
      body_html: page_record.body_html,
      modified_at: page_record.modified_at,
      published_at: page_record.published_at,
      pinned_at: page_record.pinned_at,
      space: SpaceRepository.new.to_model(space_record: page_record.space_record.not_nil!),
      topic: TopicRepository.new.to_model(topic_record: page_record.topic_record.not_nil!),
      can_update:
    )
  end

  sig do
    params(page_records: T.any(T::Array[PageRecord], PageRecord::PrivateCollectionProxy))
      .returns(T::Array[Page])
  end
  def to_models(page_records:)
    page_records.map { to_model(page_record: _1) }
  end
end
