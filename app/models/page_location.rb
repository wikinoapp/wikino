# typed: strict
# frozen_string_literal: true

class PageLocation < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :key, PageLocationKey
  const :topic, Topic
  const :page, Page

  sig { params(current_space: SpaceRecord, keys: T::Array[PageLocationKey]).returns(T::Array[PageLocation]) }
  def self.build_with_keys(current_space:, keys:)
    topics = current_space.topics.where(name: keys.map(&:topic_name).uniq)
    pages = current_space.pages.where(title: keys.map(&:page_title).uniq)

    keys.each_with_object([]) do |key, ary|
      topic = topics.find { |topic| topic.name == key.topic_name }
      page = pages.find { |page| page.topic_id == topic&.id && page.title == key.page_title }

      if topic && page
        ary << new(key:, topic:, page:)
      end
    end
  end
end
