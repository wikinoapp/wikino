# typed: strict
# frozen_string_literal: true

class Markup
  extend T::Sig

  sig { params(current_topic: Topic).void }
  def initialize(current_topic:)
    @current_topic = current_topic
  end

  sig { params(text: String).returns(String) }
  def render_html(text:)
    doc = Commonmarker.parse(text, options: {
      parse: {smart: true},
      render: {hardbreaks: false}
    })
    page_locations = PageLocation.scan_text(text, current_topic:)
    pages_with_page_location = Page.pages_with_page_location(space: current_topic.space, page_locations:)

    doc.walk do |node|
      if node.type == :text
        location_keys = node.string_content.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)

        location_keys.each do |location_key|
          page_location = PageLocation.from_location_key(location_key:, current_topic:)

          if page_location && pages_with_page_location[page_location]
            node.string_content = pages_with_page_location[page_location].id.to_s
          end
        end
      end
    end

    doc.to_html
  end

  sig { returns(Topic) }
  attr_reader :current_topic
  private :current_topic
end
