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
    return "" if text.empty?

    location_keys = PageLocationKey.scan_text(text:, current_topic:)
    page_locations = PageLocation.build_with_keys(current_space: current_topic.space.not_nil!, keys: location_keys)

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
            current_topic:,
            page_locations:
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
