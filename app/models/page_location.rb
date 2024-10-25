# typed: strict
# frozen_string_literal: true

class PageLocation < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :key, PageLocationKey
  const :topic, Topic
  const :page, Page

  sig { params(current_space: Space, keys: T::Array[PageLocationKey]).returns(T::Array[PageLocation]) }
  def self.build_with_keys(current_space:, keys:)
    topics = current_space.topics.where(name: keys.map(&:topic_name))
    pages = current_space.pages.where(title: keys.map(&:page_title))

    keys.each_with_object([]) do |key, ary|
      topic = topics.find { |topic| topic.name == key.topic_name }
      page = pages.find { |page| page.topic == topic && page.title == key.page_title }

      if page
        ary << new(key:, topic:, page:)
      end
    end
  end
end
