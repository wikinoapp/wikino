# typed: strict
# frozen_string_literal: true

require "cgi"
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

    # 単一テキストの場合はバッチ処理を使用
    results = render_html_batch(texts: [text])
    results.first || ""
  end

  sig { params(texts: T::Array[String]).returns(T::Array[String]) }
  def render_html_batch(texts:)
    return [] if texts.empty?

    # 全てのテキストから一括でlocation_keysを抽出
    all_location_keys = []
    text_to_keys_mapping = {}

    texts.each do |text|
      keys = PageLocationKey.scan_text(text:, current_topic:)
      text_to_keys_mapping[text] = keys
      all_location_keys.concat(keys)
    end

    # 重複を除いて一括でpage_locationsを取得
    unique_keys = all_location_keys.uniq { |key| "#{key.topic_name}/#{key.page_title}" }
    all_page_locations = if unique_keys.empty?
      []
    else
      PageLocationRepository.new.to_models_by_keys(current_space:, keys: unique_keys)
    end

    # 各テキストを処理
    texts.map do |text|
      next "" if text.empty?

      # このテキストに関連するpage_locationsを抽出
      text_keys = text_to_keys_mapping[text] || []
      relevant_page_locations = all_page_locations.select do |location|
        text_keys.any? do |key|
          location.key.topic_name == key.topic_name && location.key.page_title == key.page_title
        end
      end

      # HTMLブロックとMarkdownの混在を処理
      # 単独行のimgタグの前にゼロ幅非接合子を追加してインラインHTMLとして処理
      processed_text = text.gsub(/^(<img[^>]*>)\n(\*[^*]+\*)$/) { "\u200C\n#{$1}\n#{$2}" }

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
              page_locations: relevant_page_locations
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
      result = pipeline.call(processed_text)

      # ゼロ幅非接合子と余分な改行を削除
      output = result[:output].to_s
      output = output.gsub(/^<p>‌<br \/>/, "<p>").gsub(/‌/, "")

      # 単独の画像リンク（a.wikino-attachment-image-link）をp要素で囲む
      # p要素で囲まれていない、完全に独立した画像リンクのみを対象とする
      # 連続する改行の間にあり、前後に他のHTML要素がない場合のみp要素で囲む
      output.gsub(/(?<=\n|^)(<a\s+[^>]*class="wikino-attachment-image-link"[^>]*>.*?<\/a>)(?=\n|$)/m) do
        link = $1

        # この画像リンクが既にp要素内にあるかチェック
        # または直後にbr要素やem要素などのインライン要素が続くかチェック
        if output.match?(/<p[^>]*>[^<]*#{Regexp.escape(link)}[^<]*<\/p>/m) ||
            output.match?(/#{Regexp.escape(link)}\s*\n\s*<br/m) ||
            output.match?(/#{Regexp.escape(link)}\s*\n\s*<em>/m)
          link  # そのまま
        else
          "<p>\n  #{link}\n</p>"  # p要素で囲む
        end
      end
    end
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
