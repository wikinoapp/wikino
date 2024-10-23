# typed: strict
# frozen_string_literal: true

class PageLocation < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :topic_name, String
  const :page_title, String

  sig { params(location_key: String, current_topic: Topic).returns(T.nilable(PageLocation)) }
  def self.from_location_key(location_key:, current_topic:)
    topic_name, page_title = location_key.split("/", 2)

    if !topic_name.nil? && !page_title.nil?
      new(topic_name:, page_title:)
    elsif !topic_name.nil? && page_title.nil?
      new(topic_name: current_topic.name, page_title: topic_name)
    end
  end

  sig { params(text: String, current_topic: Topic).returns(T::Array[PageLocation]) }
  def self.scan_text(text, current_topic:)
    location_keys = text.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)

    location_keys.each_with_object([]) do |location_key, ary|
      page_location = from_location_key(location_key:, current_topic:)
      ary << page_location if page_location
    end
  end
end
