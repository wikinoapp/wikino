# typed: strict
# frozen_string_literal: true

class PageLocation < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :topic, Topic
  const :page_title, String

  sig { params(text: String, current_topic: Topic).returns(T::Array[PageLocation]) }
  def self.scan_text(text:, current_topic:)
    location_keys = text.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)

    locations = location_keys.each_with_object([]) do |location_key, ary|
      topic_name, page_title = location_key.split("/", 2)

      if !topic_name.nil? && !page_title.nil?
        ary << {topic_name:, page_title:}
      elsif !topic_name.nil? && page_title.nil?
        ary << {topic_name: current_topic.name, page_title: topic_name}
      end
    end

    current_space = current_topic.space
    topics = current_space.topics.where(name: locations.pluck(:topic_name))

    locations.each_with_object([]) do |location, ary|
      topic = topics.find { |topic| topic.name == location[:topic_name] }

      if topic
        ary << new(topic:, page_title: location[:page_title])
      end
    end
  end
end
