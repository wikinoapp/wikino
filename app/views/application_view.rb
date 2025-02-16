# typed: strict
# frozen_string_literal: true

class ApplicationView < ViewComponent::Base
  extend T::Sig

  sig { returns(T::Hash[Symbol, T.untyped]) }
  def default_meta_tags
    {
      reverse: true,
      site: "Wikino",
      separator: " |",
      description: I18n.t("meta.description.default"),
      canonical: "#{request.protocol}#{request.host_with_port}#{request.path}",
      og: {
        title: meta_tags.full_title(site: "Wikino", separator: " |", reverse: true),
        type: "website",
        url: request.url,
        description: I18n.t("meta.description.default"),
        site_name: "Wikino",
        image: "#{request.protocol}#{request.host_with_port}/og-image.png",
        locale: (I18n.locale == :ja) ? "ja_JP" : "en_US"
      }
    }
  end
end
