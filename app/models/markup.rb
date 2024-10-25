# typed: strict
# frozen_string_literal: true

require "html_pipeline"
require "html_pipeline/convert_filter/markdown_filter"

class Markup
  extend T::Sig

  sig { params(current_topic: Topic).void }
  def initialize(current_topic:)
    @current_topic = current_topic
  end

  sig { params(text: String).returns(String) }
  def render_html(text:)
    page_locations = PageLocation.scan_text(text:, current_topic:)
    pages = Page.all_from_page_locations(page_locations:).preload(:topic)

    pipeline = HTMLPipeline.new(
      text_filters: [],
      convert_filter: HTMLPipeline::ConvertFilter::MarkdownFilter.new(
        context: {
          markdown: {
            parse: {smart: true},
            render: {hardbreaks: false}
          }
        }
      ),
      sanitization_config: HTMLPipeline::SanitizationFilter::DEFAULT_CONFIG,
      node_filters: [
        MarkupFilters::PageLinkFilter.new(
          context: {
            pages:
          }
        )
      ]
    )
    result = pipeline.call(text)

    result[:output].to_s
  end

  sig { returns(Topic) }
  attr_reader :current_topic
  private :current_topic
end
