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
      node_filters: []
    )
    result = pipeline.call(text)

    result[:output].to_s
  end

  sig { returns(Topic) }
  attr_reader :current_topic
  private :current_topic
end
