# typed: strict
# frozen_string_literal: true

module MarkupFilters
  class PageLinkFilter < HTMLPipeline::NodeFilter
    extend T::Sig

    # @override
    def initialize(context: {}, result: {})
      super
      @page_locations = T.let(context[:page_locations], T::Array[PageLocation])
      @current_topic = T.let(context[:current_topic], Topic)
    end

    # @override
    def validate
      needs(:current_topic, :page_locations)
    end

    # @override
    def selector
      Selma::Selector.new(
        match_text_within: "*",
        ignore_text_within: %w[a code pre script style]
      )
    end

    # @override
    def handle_text_chunk(text)
      content = text.to_s

      return if !content.include?("[[") || !content.include?("]]")

      location_keys = PageLocationKey.scan_text(text: content, current_topic:)
      location_keys.each do |location_key|
        page_location = page_locations.find { |page_location| page_location.key == location_key }

        if page_location
          text.replace("#{content} (##{page_location.page.number})", as: :html)
        end
      end
    end

    sig { returns(Topic) }
    attr_reader :current_topic
    private :current_topic

    sig { returns(T::Array[PageLocation]) }
    attr_reader :page_locations
    private :page_locations
  end
end
