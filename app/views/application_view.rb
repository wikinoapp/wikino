# typed: strict
# frozen_string_literal: true

class ApplicationView < ViewComponent::Base
  extend T::Sig

  sig { params(site: T::Boolean).returns(T::Hash[Symbol, T.untyped]) }
  def default_meta_tags(site: true)
    attrs = {
      reverse: true,
      description: I18n.t("meta.description.default"),
      canonical: "#{request.protocol}#{request.host_with_port}#{request.path}",
      og: {
        title: :full_title,
        type: "website",
        url: request.url,
        description: I18n.t("meta.description.default"),
        site_name: "Wikino",
        image: "#{request.protocol}#{request.host_with_port}/og-image.png",
        locale: (I18n.locale == :ja) ? "ja_JP" : "en_US"
      }
    }

    if site
      attrs[:site] = "Wikino"
      attrs[:separator] = " |"
    end

    attrs
  end
end
