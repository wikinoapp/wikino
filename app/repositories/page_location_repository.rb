# typed: strict
# frozen_string_literal: true

class PageLocationRepository < ApplicationRepository
  sig { params(current_space: SpaceRecord, keys: T::Array[PageLocationKey]).returns(T::Array[PageLocation]) }
  def to_models(current_space:, keys:)
    topic_records = current_space.topic_records.where(name: keys.map(&:topic_name).uniq)
    page_records = current_space.page_records.where(title: keys.map(&:page_title).uniq)

    keys.each_with_object([]) do |key, ary|
      topic_record = topic_records.find { |topic_record| topic_record.name == key.topic_name }
      page_record = page_records.find do |page_record|
        page_record.topic_id == topic_record&.id && page_record.title == key.page_title
      end

      if topic_record && page_record
        ary << PageLocation.new(
          key:,
          topic: TopicRepository.new.to_model(topic_record:),
          page: PageRepository.new.to_model(page_record:)
        )
      end
    end
  end
end
