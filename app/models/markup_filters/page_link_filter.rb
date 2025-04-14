# typed: strict
# frozen_string_literal: true

module MarkupFilters
  class PageLinkFilter < HTMLPipeline::NodeFilter
    extend T::Sig

    # @override
    sig { params(context: T::Hash[Symbol, T.untyped], result: T::Hash[Symbol, T.untyped]).void }
    def initialize(context: {}, result: {})
      super
      @current_topic = T.let(context[:current_topic], TopicRecord)
      @page_locations = T.let(context[:page_locations], T::Array[PageLocation])
    end

    # @override
    sig { void }
    def validate
      needs(:current_topic, :page_locations)
    end

    # @override
    sig { returns(Selma::Selector) }
    def selector
      Selma::Selector.new(
        match_text_within: "*",
        ignore_text_within: %w[a code pre script style]
      )
    end

    # @override
    sig { params(text_chunk: Selma::HTML::TextChunk).void }
    def handle_text_chunk(text_chunk)
      text = CGI.unescapeHTML(text_chunk.to_s)

      return if !text.include?("[[") || !text.include?("]]")

      location_keys = PageLocationKey.scan_text(text:, current_topic:)
      location_keys.each do |location_key|
        page_location = page_locations.find { |page_location| page_location.key == location_key }

        if page_location
          text.gsub!(
            /\[\[#{Regexp.escape(location_key.raw)}\]\]/,
            view_context.render(PageLinkComponent.new(current_space: current_topic.space.not_nil!, page_location:))
          )
          text_chunk.replace(text, as: :html)
        end
      end
    end

    sig { returns(TopicRecord) }
    attr_reader :current_topic
    private :current_topic

    sig { returns(T::Array[PageLocation]) }
    attr_reader :page_locations
    private :page_locations

    sig { returns(ActionView::Base) }
    private def view_context
      ApplicationController.new.view_context
    end
  end
end
