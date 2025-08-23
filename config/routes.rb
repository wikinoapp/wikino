# typed: false
# frozen_string_literal: true

Rails.application.routes.draw do
  mount MissionControl::Jobs::Engine, at: "/jobs"

  if Rails.env.development?
    mount LetterOpenerWeb::Engine, at: "/letter_opener"
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/_test/session",                                                  via: :post,   as: :test_session,                                 to: "test/sessions/create#call" if Rails.env.test?
  match "/_test/attachments/presign",                                      via: :post,   as: :test_attachment_presign,                      to: "test/attachments/presigns/create#call" if Rails.env.test?
  match "/_test/attachments/upload",                                       via: :put,    as: :test_attachment_upload,                       to: "test/attachments/uploads/create#call" if Rails.env.test?
  match "/@:atname",                                                       via: :get,    as: :profile,                                      to: "profiles/show#call"
  match "/accounts",                                                       via: :post,   as: :account_list,                                 to: "accounts/create#call"
  match "/accounts/new",                                                   via: :get,    as: :new_account,                                  to: "accounts/new#call"
  match "/attachments/:attachment_id",                                     via: :get,    as: :attachment,                                   to: "attachments/show#call"
  match "/attachments/signed_urls",                                        via: :post,   as: :attachment_signed_url_list,                   to: "attachments/signed_urls/create#call"
  match "/email_confirmation",                                             via: :patch,  as: :email_confirmation,                           to: "email_confirmations/update#call"
  match "/email_confirmation",                                             via: :post,                                                      to: "email_confirmations/create#call"
  match "/email_confirmation/edit",                                        via: :get,    as: :edit_email_confirmation,                      to: "email_confirmations/edit#call"
  match "/home",                                                           via: :get,    as: :home,                                         to: "home/show#call"
  match "/manifest",                                                       via: :get,    as: :manifest,                                     to: "manifests/show#call"
  match "/password_reset",                                                 via: :get,    as: :password_reset,                               to: "password_resets/new#call"
  match "/password_reset",                                                 via: :post,                                                      to: "password_resets/create#call"
  match "/password",                                                       via: :patch,  as: :password,                                     to: "passwords/update#call"
  match "/password/edit",                                                  via: :get,    as: :edit_password,                                to: "passwords/edit#call"
  match "/privacy",                                                        via: :get,    as: :privacy,                                      to: redirect("https://wikino.app/s/wikino/pages/42")
  match "/s/:space_identifier",                                            via: :get,    as: :space,                                        to: "spaces/show#call"
  match "/s/:space_identifier/atom",                                       via: :get,    as: :atom,                                         to: "atom/show#call"
  match "/s/:space_identifier/attachments",                                via: :post,   as: :attachment_list,                              to: "attachments/create#call"
  match "/s/:space_identifier/attachments/presign",                        via: :post,   as: :attachment_presign,                           to: "attachments/presigns/create#call"
  match "/s/:space_identifier/bulk_restored_pages",                        via: :post,   as: :bulk_restored_page_list,                      to: "bulk_restored_pages/create#call"
  match "/s/:space_identifier/page_locations",                             via: :get,    as: :page_location_list,                           to: "page_locations/index#call",                   page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number",                         via: :get,    as: :page,                                         to: "pages/show#call",                             page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number",                         via: :patch,                                                     to: "pages/update#call",                           page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/backlinks",               via: :post,   as: :page_backlink_list,                           to: "backlinks/index#call",                        page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/draft_page",              via: :patch,  as: :draft_page,                                   to: "draft_pages/update#call",                     page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/edit",                    via: :get,    as: :edit_page,                                    to: "pages/edit#call",                             page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/links",                   via: :post,   as: :page_link_list,                               to: "links/index#call",                            page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/trash",                   via: :post,   as: :trashed_page,                                 to: "trashed_pages/create#call",                   page_number: /\d+/
  match "/s/:space_identifier/settings",                                   via: :get,    as: :space_settings,                               to: "spaces/settings/show#call"
  match "/s/:space_identifier/settings/deletion",                          via: :post,   as: :space_settings_deletion,                      to: "spaces/settings/deletions/create#call"
  match "/s/:space_identifier/settings/deletion/new",                      via: :get,    as: :space_settings_new_deletion,                  to: "spaces/settings/deletions/new#call"
  match "/s/:space_identifier/settings/exports",                           via: :post,   as: :space_settings_export_list,                   to: "spaces/settings/exports/create#call"
  match "/s/:space_identifier/settings/exports/:export_id",                via: :get,    as: :space_settings_export,                        to: "spaces/settings/exports/show#call",           export_id: /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/
  match "/s/:space_identifier/settings/exports/:export_id/download",       via: :get,    as: :space_settings_download_export,               to: "spaces/settings/exports/downloads/show#call", export_id: /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/
  match "/s/:space_identifier/settings/exports/new",                       via: :get,    as: :space_settings_new_exports,                   to: "spaces/settings/exports/new#call"
  match "/s/:space_identifier/settings/general",                           via: :get,    as: :space_settings_general,                       to: "spaces/settings/general/show#call"
  match "/s/:space_identifier/settings/general",                           via: :patch,                                                     to: "spaces/settings/general/update#call"
  match "/s/:space_identifier/settings/attachments",                       via: :get,    as: :space_settings_attachments,                   to: "spaces/settings/attachments/index#call"
  match "/s/:space_identifier/settings/attachments/:attachment_id",        via: :delete, as: :space_settings_attachment,                    to: "spaces/settings/attachments/destroy#call"
  match "/s/:space_identifier/topics",                                     via: :post,   as: :topic_list,                                   to: "topics/create#call"
  match "/s/:space_identifier/topics/:topic_number",                       via: :get,    as: :topic,                                        to: "topics/show#call",                            topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/pages/new",             via: :get,    as: :new_page,                                     to: "pages/new#call",                              topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/settings",              via: :get,    as: :topic_settings,                               to: "topics/settings/show#call",                   topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/settings/deletion",     via: :post,   as: :topic_settings_deletion,                      to: "topics/settings/deletions/create#call",       topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/settings/deletion/new", via: :get,    as: :topic_settings_new_deletion,                  to: "topics/settings/deletions/new#call",          topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/settings/general",      via: :get,    as: :topic_settings_general,                       to: "topics/settings/general/show#call",           topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/settings/general",      via: :patch,                                                     to: "topics/settings/general/update#call",         topic_number: /\d+/
  match "/s/:space_identifier/topics/new",                                 via: :get,    as: :new_topic,                                    to: "topics/new#call"
  match "/s/:space_identifier/trash",                                      via: :get,    as: :trash,                                        to: "trash/show#call"
  match "/search",                                                         via: :get,    as: :search,                                       to: "search/show#call"
  match "/settings",                                                       via: :get,    as: :settings,                                     to: "settings/show#call"
  match "/settings/account/deletion",                                      via: :post,   as: :settings_account_deletion,                    to: "settings/account/deletions/create#call"
  match "/settings/account/deletion/new",                                  via: :get,    as: :settings_account_new_deletion,                to: "settings/account/deletions/new#call"
  match "/settings/email",                                                 via: :get,    as: :settings_email,                               to: "settings/emails/show#call"
  match "/settings/email",                                                 via: :patch,                                                     to: "settings/emails/update#call"
  match "/settings/profile",                                               via: :get,    as: :settings_profile,                             to: "settings/profiles/show#call"
  match "/settings/profile",                                               via: :patch,                                                     to: "settings/profiles/update#call"
  match "/settings/two_factor_auth",                                       via: :delete, as: :settings_two_factor_auth,                     to: "settings/two_factor_auths/destroy#call"
  match "/settings/two_factor_auth",                                       via: :get,                                                       to: "settings/two_factor_auths/show#call"
  match "/settings/two_factor_auth",                                       via: :post,                                                      to: "settings/two_factor_auths/create#call"
  match "/settings/two_factor_auth/new",                                   via: :get,    as: :settings_new_two_factor_auth,                 to: "settings/two_factor_auths/new#call"
  match "/settings/two_factor_auth/recovery_codes",                        via: :get,    as: :settings_two_factor_auth_recovery_code_list,  to: "settings/two_factor_auths/recovery_codes/show#call"
  match "/settings/two_factor_auth/recovery_codes",                        via: :post,                                                      to: "settings/two_factor_auths/recovery_codes/create#call"
  match "/sign_in",                                                        via: :get,    as: :sign_in,                                      to: "sign_in/show#call"
  match "/sign_in/two_factor/new",                                         via: :get,    as: :sign_in_new_two_factor,                       to: "sign_in/two_factors/new#call"
  match "/sign_in/two_factor",                                             via: :post,   as: :sign_in_two_factor,                           to: "sign_in/two_factors/create#call"
  match "/sign_in/two_factor/recovery/new",                                via: :get,    as: :sign_in_two_factor_new_recovery,              to: "sign_in/two_factors/recoveries/new#call"
  match "/sign_in/two_factor/recovery",                                    via: :post,   as: :sign_in_two_factor_recovery,                  to: "sign_in/two_factors/recoveries/create#call"
  match "/sign_up",                                                        via: :get,    as: :sign_up,                                      to: "sign_up/show#call"
  match "/spaces",                                                         via: :post,   as: :space_list,                                   to: "spaces/create#call"
  match "/spaces/new",                                                     via: :get,    as: :new_space,                                    to: "spaces/new#call"
  match "/terms",                                                          via: :get,    as: :terms,                                        to: redirect("https://wikino.app/s/wikino/pages/41")
  match "/user_session",                                                   via: :delete, as: :user_session,                                 to: "user_sessions/destroy#call"
  match "/user_session",                                                   via: :post,                                                      to: "user_sessions/create#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
