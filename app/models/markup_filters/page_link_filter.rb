# typed: strict
# frozen_string_literal: true

module MarkupFilters
  class PageLinkFilter < HTMLPipeline::NodeFilter
    extend T::Sig

    # @override
    def initialize(context: {}, result: {})
      super
      @pages = T.let(context[:pages], Page::PrivateRelation)
    end

    # @override
    def validate
      needs(:pages)
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

      location_keys = content.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)
      location_keys.each do |location_key|
        topic_name, page_title = location_key.split("/", 2)

        page = nil
        if !topic_name.nil? && !page_title.nil?
          page = pages.find { |page| page.topic.name == topic_name && page.title == page_title }
        elsif !topic_name.nil? && page_title.nil?
          page = pages.find { |page| page.topic.name == current_topic.name && page.title == topic_name }
        end

        if page
          text.replace("#{content} (##{page.number})")
        end
      end
    end

    sig { returns(Topic) }
    attr_reader :current_topic
    private :current_topic

    sig { returns(T::Array[Page]) }
    attr_reader :pages
    private :pages
  end
end
