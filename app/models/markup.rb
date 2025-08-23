# typed: strict
# frozen_string_literal: true

require "selma"
require "html_pipeline"
require "html_pipeline/convert_filter/markdown_filter"

class Markup
  extend T::Sig

  sig { params(current_topic: Topic, current_space: Space, current_space_member: T.nilable(SpaceMember)).void }
  def initialize(current_topic:, current_space:, current_space_member: nil)
    @current_topic = current_topic
    @current_space = current_space
    @current_space_member = current_space_member
  end

  sig { params(text: String).returns(String) }
  def render_html(text:)
    return "" if text.empty?

    location_keys = PageLocationKey.scan_text(text:, current_topic:)
    page_locations = PageLocationRepository.new.to_models_by_keys(current_space:, keys: location_keys)

    pipeline = HTMLPipeline.new(
      text_filters: [],
      convert_filter: HTMLPipeline::ConvertFilter::MarkdownFilter.new(
        context: {
          markdown: {
            parse: {smart: false, html: true},  # HTMLタグの解析を有効化
            render: {hardbreaks: true, unsafe: true}  # HTMLタグのレンダリングを有効化
          }
        }
      ),
      sanitization_config:,
      node_filters: [
        Markup::PageLinkFilter.new(
          context: {
            current_topic:,
            page_locations:
          }
        ),
        Markup::AttachmentFilter.new(
          context: {
            current_space:,
            current_space_member:
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

  sig { returns(Space) }
  attr_reader :current_space
  private :current_space

  sig { returns(T.nilable(SpaceMember)) }
  attr_reader :current_space_member
  private :current_space_member

  sig { returns(T::Hash[T.any(Symbol, String), T.untyped]) }
  private def sanitization_config
    default_sanitization_config = HTMLPipeline::SanitizationFilter::DEFAULT_CONFIG

    Selma::Sanitizer::Config.merge(
      default_sanitization_config,
      {
        elements: default_sanitization_config[:elements] + [
          # タスクリスト記法 (`- [ ]`) のために許可する
          "input"
        ],
        attributes: Selma::Sanitizer::Config.merge(
          default_sanitization_config[:attributes],
          {
            "img" => (default_sanitization_config[:attributes]["img"] || []) + ["width", "height"]
          }
        )
      }
    )
  end
end
