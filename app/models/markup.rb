# typed: strict
# frozen_string_literal: true

require "selma"
require "html_pipeline"
require "html_pipeline/convert_filter/markdown_filter"

class Markup
  extend T::Sig

  sig { params(current_topic: TopicRecord).void }
  def initialize(current_topic:)
    @current_topic = current_topic
  end

  sig { params(text: String).returns(String) }
  def render_html(text:)
    return "" if text.empty?

    location_keys = PageLocationKey.scan_text(text:, current_topic:)
    page_locations = PageLocationRepository.new.to_models(current_space: current_topic.space_record.not_nil!, keys: location_keys)

    pipeline = HTMLPipeline.new(
      text_filters: [],
      convert_filter: HTMLPipeline::ConvertFilter::MarkdownFilter.new(
        context: {
          markdown: {
            parse: {smart: false},
            render: {hardbreaks: true}
          }
        }
      ),
      sanitization_config:,
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

  sig { returns(TopicRecord) }
  attr_reader :current_topic
  private :current_topic

  sig { returns(T::Hash[T.any(Symbol, String), T.untyped]) }
  private def sanitization_config
    default_sanitization_config = HTMLPipeline::SanitizationFilter::DEFAULT_CONFIG

    Selma::Sanitizer::Config.merge(
      default_sanitization_config,
      {
        elements: default_sanitization_config[:elements] + [
          # タスクリスト記法 (`- [ ]`) のために許可する
          "input"
        ]
      }
    )
  end
end
