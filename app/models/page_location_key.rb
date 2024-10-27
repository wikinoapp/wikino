# typed: strict
# frozen_string_literal: true

class PageLocationKey < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  # 例: "topic_name/page_title"
  const :raw, String

  const :topic_name, String
  const :page_title, String

  sig { params(text: String, current_topic: Topic).returns(T::Array[PageLocationKey]) }
  def self.scan_text(text:, current_topic:)
    location_keys = text.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)

    location_keys.each_with_object([]) do |location_key, ary|
      topic_name, page_title = location_key.split("/", 2)

      if !topic_name.nil? && !page_title.nil?
        ary << new(raw: location_key, topic_name:, page_title:)
      elsif !topic_name.nil? && page_title.nil?
        ary << new(raw: location_key, topic_name: current_topic.name, page_title: topic_name)
      end
    end
  end
end
